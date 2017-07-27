package artifactory

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	gitconfig "gopkg.in/src-d/go-git.v4/plumbing/format/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
)

type GitLfsCleanService struct {
	client *helpers.JfrogHttpClient
	ArtDetails *auth.ArtifactoryAuthConfiguration
	DryRun     bool
}

func (glc *GitLfsCleanService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return glc.ArtDetails
}

func (glc *GitLfsCleanService) SetArtifactoryDetails(art *auth.ArtifactoryAuthConfiguration) {
	glc.ArtDetails = art
}

func (glc *GitLfsCleanService) IsDryRun() bool {
	return glc.DryRun
}

func NewGitLfsCleanService(client *helpers.JfrogHttpClient) *GitLfsCleanService {
	return &GitLfsCleanService{client: client}
}

func (glc *GitLfsCleanService) GetRedundantGitLfsFiles(gitLfsCleanParams GitLfsCleanParams) ([]utils.AqlSearchResultItem, error) {
	var err error
	repo := gitLfsCleanParams.GetRepo()
	gitPath := gitLfsCleanParams.GetGitPath()
	if gitPath == "" {
		gitPath, err = os.Getwd()
		if err != nil {
			return nil, cliutils.CheckError(err)
		}
	}
	if len(repo) <= 0 {
		repo, err = detectRepo(gitPath, glc.ArtDetails.Url)
		if err != nil {
			return nil, err
		}
	}
	log.Info("Searching files from Artifactory repository", repo, "...")
	refsRegex := getRefsRegex(gitLfsCleanParams.GetRef())
	artifactoryLfsFiles, err := glc.searchLfsFilesInArtifactory(repo)
	if err != nil {
		return nil, cliutils.CheckError(err)
	}
	log.Info("Collecting files to preserve from Git references matching the pattern", gitLfsCleanParams.GetRef(), "...")
	gitLfsFiles, err := getLfsFilesFromGit(gitPath, refsRegex)
	if err != nil {
		return nil, cliutils.CheckError(err)
	}
	filesToDelete := findFilesToDelete(artifactoryLfsFiles, gitLfsFiles)
	log.Info("Found", len(gitLfsFiles), "files to keep, and", len(filesToDelete), "to clean")
	return filesToDelete, nil
}

func findFilesToDelete(artifactoryLfsFiles []utils.AqlSearchResultItem, gitLfsFiles map[string]struct{}) []utils.AqlSearchResultItem {
	results := make([]utils.AqlSearchResultItem, 0, len(artifactoryLfsFiles))
	for _, file := range artifactoryLfsFiles {
		if _, keepFile := gitLfsFiles[file.Name]; !keepFile {
			results = append(results, file)
		}
	}
	return results
}

func lfsConfigUrlExtractor (conf *gitconfig.Config) (*url.URL, error) {
	return url.Parse(conf.Section("lfs").Option("url"))
}

func configLfsUrlExtractor(conf *gitconfig.Config) (*url.URL, error) {
	return url.Parse(conf.Section("remote").Subsection("origin").Option("lfsurl"))
}

func detectRepo(gitPath, rtUrl string) (string, error) {
	repo, err := extractRepo(gitPath, ".lfsconfig", rtUrl, lfsConfigUrlExtractor)
	if err == nil {
		return repo, nil
	}
	errMsg1 := fmt.Sprintln("Cannot detect Git LFS repository from .lfsconfig: %s", err)
	repo, err = extractRepo(gitPath, ".git/config", rtUrl, configLfsUrlExtractor)
	if err == nil {
		return repo, nil
	}
	errMsg2 := fmt.Sprintln("Cannot detect Git LFS repository from .git/config: %s", err)
	suggestedSolution := "You may want to try passing the --repo option manually"
	return "", cliutils.CheckError(fmt.Errorf("%s%s%s", errMsg1, errMsg2, suggestedSolution))
}

func extractRepo(gitPath, configFile, rtUrl string, lfsUrlExtractor lfsUrlExtractorFunc) (string, error) {
	lfsUrl, err := getLfsUrl(gitPath, configFile, lfsUrlExtractor)
	if err != nil {
		return "", err
	}
	artifactoryConfiguredUrl, err := url.Parse(rtUrl)
	if err != nil {
		return "", err
	}
	if artifactoryConfiguredUrl.Scheme != lfsUrl.Scheme || artifactoryConfiguredUrl.Host != lfsUrl.Host {
		return "", fmt.Errorf("Configured Git LFS URL %q does not match provided URL %q", lfsUrl.String(), artifactoryConfiguredUrl.String())
	}
	artifactoryConfiguredUrlPath := path.Clean("/" + artifactoryConfiguredUrl.Path + "/api/lfs") + "/"
	lfsUrlPath := path.Clean(lfsUrl.Path)
	if strings.HasPrefix(lfsUrlPath, artifactoryConfiguredUrlPath) {
		return lfsUrlPath[len(artifactoryConfiguredUrlPath):], nil
	}
	return "", fmt.Errorf("Configured Git LFS URL %q does not match provided URL %q", lfsUrl.String(), artifactoryConfiguredUrl.String())
}

type lfsUrlExtractorFunc func(conf *gitconfig.Config) (*url.URL, error)

func getLfsUrl(gitPath, configFile string, lfsUrlExtractor lfsUrlExtractorFunc) (*url.URL, error) {
	var lfsUrl *url.URL
	lfsConf, err := os.Open(path.Join(gitPath, configFile))
	if err != nil {
		return nil, cliutils.CheckError(err)
	}
	defer lfsConf.Close()
	conf := gitconfig.New()
	err = gitconfig.NewDecoder(lfsConf).Decode(conf)
	if err != nil {
		return nil, cliutils.CheckError(err)
	}
	lfsUrl, err = lfsUrlExtractor(conf)
	return lfsUrl, cliutils.CheckError(err)
}

func getRefsRegex(refs string) string {
	replacer := strings.NewReplacer(",", "|", "\\*", ".*")
	return replacer.Replace(regexp.QuoteMeta(refs))
}

func (glc *GitLfsCleanService) searchLfsFilesInArtifactory(repo string) ([]utils.AqlSearchResultItem, error) {
	spec := &utils.File{Pattern:repo, Target:"", Props:"", Build:"", Recursive:"true", Flat:"false", Regexp:"false", IncludeDirs:"false"}
	return utils.AqlSearchDefaultReturnFields(spec, glc, glc.client)
}

func getLfsFilesFromGit(path, refMatch string) (map[string]struct{}, error) {
	// a hash set of sha2 sums, to make lookup faster later
	results := make(map[string]struct{}, 0)
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, cliutils.CheckError(err)
	}
	log.Debug("Opened Git repo at", path, "for reading")
	refs, err := repo.References()
	if err != nil {
		return nil, cliutils.CheckError(err)
	}
	// look for every Git LFS pointer file that exists in any ref (branch,
	// remote branch, tag, etc.) who's name matches the regex refMatch
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// go-git recognizes three types of refs: regular hash refs,
		// symbolic refs (e.g. HEAD), and invalid refs. We only care
		// about the first type here.
		if ref.Type() != plumbing.HashReference {
			return nil
		}
		log.Debug("Checking ref", ref.Name().String())
		match, err := regexp.MatchString(refMatch, ref.Name().String())
		if err != nil || !match {
			return cliutils.CheckError(err)
		}
		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			return cliutils.CheckError(err)
		}
		files, err := commit.Files()
		if err != nil {
			return cliutils.CheckError(err)
		}
		err = files.ForEach(func(file *object.File) error {
			return collectLfsFileFromGit(results, file)
		})
		return cliutils.CheckError(err)
	})
	return results, cliutils.CheckError(err)
}

func collectLfsFileFromGit(results map[string]struct{}, file *object.File) error {
	// A Git LFS pointer is a small file containing a sha2. Any file bigger
	// than a kilobyte is extremely unlikely to be such a pointer.
	if file.Size > 1024 {
		return nil
	}
	lines, err := file.Lines()
	if err != nil {
		return cliutils.CheckError(err)
	}
	// the line containing the sha2 we're looking for will match this regex
	regex := "^oid sha256:[[:alnum:]]{64}$"
	for _, line := range lines {
		if !strings.HasPrefix(line, "oid ") {
			continue
		}
		match, err := regexp.MatchString(regex, line)
		if err != nil || !match {
			return cliutils.CheckError(err)
		}
		result := line[strings.Index(line, ":") + 1:]
		log.Debug("Found file", result)
		results[result] = struct{}{}
		break
	}
	return nil
}

type GitLfsCleanParams interface {
	GetRef() string
	GetRepo() string
	GetGitPath() string
}

type GitLfsCleanParamsImpl struct {
	Refs    string
	Repo    string
	GitPath string
}

func (glc *GitLfsCleanParamsImpl) GetRef() string {
	return glc.Refs
}

func (glc *GitLfsCleanParamsImpl) GetRepo() string {
	return glc.Repo
}

func (glc *GitLfsCleanParamsImpl) GetGitPath() string {
	return glc.GitPath
}
