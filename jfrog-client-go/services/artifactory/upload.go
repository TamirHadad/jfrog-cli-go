package artifactory

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/httputils"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"errors"
	"github.com/gofrog/parallel"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"bytes"
	"sort"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
)

func NewUploadService(client *helpers.JfrogHttpClient) *UploadService {
	return &UploadService{client:client}
}

type UploadService struct {
	client            *helpers.JfrogHttpClient
	ArtDetails        *auth.ArtifactoryAuthConfiguration
	DryRun            bool
	Threads           int
	MinChecksumDeploy int64
}

func (us *UploadService) SetThread(threads int) {
	us.Threads = threads
}

func (us *UploadService) SetArtDetails(artDetails *auth.ArtifactoryAuthConfiguration) {
	us.ArtDetails = artDetails
}

func (us *UploadService) SetDryRun(isDryRun bool) {
	us.DryRun = isDryRun
}

func (us *UploadService) setMinChecksumDeploy(minChecksumDeploy int64) {
	us.MinChecksumDeploy = minChecksumDeploy
}

func (us *UploadService) UploadFiles(uploadParams UploadParams) ([]utils.ArtifactsBuildInfo, int, int, error) {
	uploadSummery := uploadResult{
		UploadCount: make([]int, us.Threads),
		TotalCount: make([]int, us.Threads),
		BuildInfoArtifacts: make([][]utils.ArtifactsBuildInfo, us.Threads),
	}
	artifactHandlerFunc := us.createArtifactHandlerFunc(&uploadSummery, uploadParams)
	producerConsumer := parallel.NewBounedRunner(us.Threads, true)
	errorsQueue := utils.NewErrorsQueue(1)
	us.prepareUploadTasks(producerConsumer, uploadParams, artifactHandlerFunc, errorsQueue)
	return us.performUploadTasks(producerConsumer, &uploadSummery, errorsQueue)
}

func (us *UploadService) prepareUploadTasks(producer parallel.Runner, uploadParams UploadParams, artifactHandlerFunc artifactContext, errorsQueue *utils.ErrorsQueue)  {
	go func() {
		collectFilesForUpload(uploadParams, producer, artifactHandlerFunc, errorsQueue)
	}()
}

func (us *UploadService) performUploadTasks(consumer parallel.Runner, uploadSummery *uploadResult, errorsQueue *utils.ErrorsQueue) (buildInfoArtifacts []utils.ArtifactsBuildInfo, totalUploaded, totalFailed int, err error) {
	// Blocking until we finish consuming for some reason
	consumer.Run()
	if e := errorsQueue.GetError(); e != nil {
		err = e
		return
	}
	totalUploaded = sumIntArray(uploadSummery.UploadCount)
	totalUploadAttempted := sumIntArray(uploadSummery.TotalCount)

	log.Info("Uploaded", strconv.Itoa(totalUploaded), "artifacts.")
	totalFailed = totalUploadAttempted - totalUploaded
	if totalFailed > 0 {
		log.Error("Failed uploading", strconv.Itoa(totalFailed), "artifacts.")
	}
	buildInfoArtifacts = toBuildInfoArtifacts(uploadSummery.BuildInfoArtifacts)
	return
}

func toBuildInfoArtifacts(artifactsBuildInfo [][]utils.ArtifactsBuildInfo) []utils.ArtifactsBuildInfo {
	var buildInfo []utils.ArtifactsBuildInfo
	for _, v := range artifactsBuildInfo {
		buildInfo = append(buildInfo, v...)
	}
	return buildInfo
}

func sumIntArray(arr []int) int {
	sum := 0
	for _, i := range arr {
		sum += i
	}
	return sum
}

func getSingleFileToUpload(rootPath, targetPath string, flat bool) cliutils.Artifact {
	var uploadPath string
	if !strings.HasSuffix(targetPath, "/") {
		uploadPath = targetPath
	} else {
		if flat {
			uploadPath, _ = fileutils.GetFileAndDirFromPath(rootPath)
			uploadPath = targetPath + uploadPath
		} else {
			uploadPath = targetPath + rootPath
			uploadPath = cliutils.TrimPath(uploadPath)
		}
	}
	symlinkPath, e := getFileSymlinkPath(rootPath)
	if e != nil {
		return cliutils.Artifact{}
	}
	return cliutils.Artifact{LocalPath: rootPath, TargetPath: uploadPath, Symlink: symlinkPath}
}

func addProps(oldProps, additionalProps string) string {
	if len(oldProps) > 0 && !strings.HasSuffix(oldProps, ";")  && len(additionalProps) > 0 {
		oldProps += ";"
	}
	return oldProps + additionalProps
}

func addSymlinkProps(artifact cliutils.Artifact, uploadParams UploadParams) (string, error) {
	artifactProps := ""
	artifactSymlink := artifact.Symlink
	if uploadParams.IsSymlink() && len(artifactSymlink) > 0 {
		sha1Property := ""
		fileInfo, err := os.Stat(artifact.LocalPath)
		if err != nil {
			return "", err
		}
		if !fileInfo.IsDir() {
			sha1, err := fileutils.CalcSha1(artifact.LocalPath)
			if err != nil {
				return "", err
			}
			sha1Property = ";" + utils.SYMLINK_SHA1 + "=" + sha1
		}
		artifactProps += utils.ARTIFACTORY_SYMLINK + "=" + artifactSymlink + sha1Property
	}
	artifactProps = addProps(uploadParams.GetProps(), artifactProps)
	return artifactProps, nil
}

func collectFilesForUpload(uploadParams UploadParams, producer parallel.Runner, artifactHandlerFunc artifactContext, errorsQueue *utils.ErrorsQueue) {
	defer producer.Done()
	if strings.Index(uploadParams.GetTarget(), "/") < 0 {
		uploadParams.SetTarget(uploadParams.GetTarget() + "/")
	}
	uploadMetaData := uploadDescriptor{}
	uploadParams.SetPattern(cliutils.ReplaceTildeWithUserHome(uploadParams.GetPattern()))
	uploadMetaData.CreateUploadDescriptor(uploadParams.GetRegexp(), uploadParams.GetFlat(), uploadParams.GetPattern())
	if uploadMetaData.Err != nil {
		errorsQueue.AddError(uploadMetaData.Err)
		return
	}
	// If the path is a single file then return it
	if !uploadMetaData.IsDir || (uploadParams.IsSymlink() && fileutils.IsPathSymlink(uploadParams.GetPattern())) {
		artifact := getSingleFileToUpload(uploadMetaData.RootPath, uploadParams.GetTarget(), uploadMetaData.IsFlat)
		props, err := addSymlinkProps(artifact, uploadParams)
		if err != nil {
			errorsQueue.AddError(err)
			return
		}
		uploadData := UploadData{Artifact: artifact, Props: props}
		task := artifactHandlerFunc(uploadData)
		producer.AddTaskWithError(task, errorsQueue.AddError)
		return
	}
	uploadParams.SetPattern(cliutils.PrepareLocalPathForUpload(uploadParams.GetPattern(), uploadMetaData.IsRegexp))
	err := collectPatternMatchingFiles(uploadParams, uploadMetaData, producer, artifactHandlerFunc, errorsQueue)
	if err != nil {
		errorsQueue.AddError(err)
		return
	}
}

func getRootPath(pattern string, isRegexp bool) (string, error){
	rootPath := cliutils.GetRootPathForUpload(pattern, isRegexp)
	if !fileutils.IsPathExists(rootPath) {
		err := cliutils.CheckError(errors.New("Path does not exist: " + rootPath))
		if err != nil {
			return "", err
		}
	}
	return rootPath, nil
}

// If filePath is path to a symlink we should return the link content e.g where the link points
func getFileSymlinkPath(filePath string) (string, error){
	fileInfo, e := os.Lstat(filePath)
	if cliutils.CheckError(e) != nil {
		return "", e
	}
	var symlinkPath = ""
	if fileutils.IsFileSymlink(fileInfo) {
		symlinkPath, e = os.Readlink(filePath)
		if cliutils.CheckError(e) != nil {
			return "", e
		}
	}
	return symlinkPath, nil
}

func getUploadPaths(isRecursiveString, rootPath string, includeDirs, isSymlink bool) ([]string, error) {
	var paths []string
	isRecursive, err := cliutils.StringToBool(isRecursiveString, true)
	if err != nil {
		return paths, err
	}
	if isRecursive {
		paths, err = fileutils.ListFilesRecursiveWalkIntoDirSymlink(rootPath, !isSymlink)
	} else {
		paths, err = fileutils.ListFiles(rootPath, includeDirs)
	}
	if err != nil {
		return paths, err
	}
	return paths, nil
}

func collectPatternMatchingFiles(uploadParams UploadParams, uploadMetaData uploadDescriptor, producer parallel.Runner, artifactHandlerFunc artifactContext, errorsQueue *utils.ErrorsQueue) error {
	r, err := regexp.Compile(uploadParams.GetPattern())
	if cliutils.CheckError(err) != nil {
		return err
	}

	paths, err := getUploadPaths(uploadParams.GetRecursive(), uploadMetaData.RootPath, uploadParams.IsIncludeDirs(), uploadParams.IsSymlink())
	if err != nil {
		return err
	}
	// Longest paths first
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	// 'foldersPaths' is a subset of the 'paths' array. foldersPaths is in use only when we need to upload folders with flat=true.
	// 'foldersPaths' will contain only the directories paths which are in the 'paths' array.
	var foldersPaths[]string
	for index, path := range paths {
		isDir, err := fileutils.IsDir(path)
		if err != nil {
			return err
		}
		isSymlinkFlow := uploadParams.IsSymlink() && fileutils.IsPathSymlink(path)
		if isDir && !uploadParams.IsIncludeDirs() && !isSymlinkFlow {
			continue
		}
		groups := r.FindStringSubmatch(path)
		size := len(groups)
		target := uploadParams.GetTarget()
		if size > 0 {
			tempPaths := paths
			tempIndex := index
			// In case we need to upload directories with flat=true, we want to avoid the creation of unnecessary paths in Artifactory.
			// To achieve this, we need to take into consideration the directories which had already been uploaded, ignoring all files paths.
			// When flat=false we take into consideration folder paths which were created implicitly by file upload
			if uploadMetaData.IsFlat && uploadParams.IsIncludeDirs() && isDir {
				foldersPaths = append(foldersPaths, path)
				tempPaths = foldersPaths
				tempIndex = len(foldersPaths) - 1
			}
			taskData := &uploadTaskData{target: target, path: path, isDir: isDir, isSymlinkFlow: isSymlinkFlow, paths: tempPaths,
				groups:                         groups, index: tempIndex, size: size, uploadParams: uploadParams, uploadMetaData: uploadMetaData,
				producer:                       producer, artifactHandlerFunc: artifactHandlerFunc, errorsQueue: errorsQueue,
			}
			createUploadTask(taskData)
		}
	}
	return nil
}

type uploadTaskData struct {
	target              string
	path                string
	isDir               bool
	isSymlinkFlow       bool
	paths               []string
	groups              []string
	index               int
	size                int
	uploadParams        UploadParams
	uploadMetaData      uploadDescriptor
	producer            parallel.Runner
	artifactHandlerFunc artifactContext
	errorsQueue         *utils.ErrorsQueue
}

func createUploadTask(taskData *uploadTaskData) error {
	for i := 1; i < taskData.size; i++ {
		group := strings.Replace(taskData.groups[i], "\\", "/", -1)
		taskData.target = strings.Replace(taskData.target, "{"+strconv.Itoa(i)+"}", group, -1)
	}
	var task parallel.TaskFunc
	taskData.target = getUploadTarget(taskData.uploadMetaData.IsFlat, taskData.path, taskData.target)
	// If case taskData.path is a symlink we get the symlink link path.
	symlinkPath, e := getFileSymlinkPath(taskData.path)
	if e != nil {
		return e
	}
	artifact := cliutils.Artifact{LocalPath: taskData.path, TargetPath: taskData.target, Symlink: symlinkPath}
	props, e := addSymlinkProps(artifact, taskData.uploadParams)
	if e != nil {
		return e
	}
	uploadData := UploadData{Artifact: artifact, Props: props}
	if taskData.isDir && taskData.uploadParams.IsIncludeDirs() && !taskData.isSymlinkFlow {
		if taskData.path != "." && (taskData.index == 0 || !utils.IsSubPath(taskData.paths, taskData.index, fileutils.GetFileSeperator())) {
			uploadData.IsDir = true
		} else {
			return nil
		}
	}
	task = taskData.artifactHandlerFunc(uploadData)
	taskData.producer.AddTaskWithError(task, taskData.errorsQueue.AddError)
	return nil
}

func getUploadTarget(isFlat bool, path, target string) string {
	if strings.HasSuffix(target, "/") {
		if isFlat {
			fileName, _ := fileutils.GetFileAndDirFromPath(path)
			target += fileName
		} else {
			target += cliutils.TrimPath(path)
		}
	}
	return target
}

func addPropsToTargetPath(targetPath, props, debConfig string) (string, error) {
	if props != "" {
		encodedProp, err := utils.EncodeParams(props)
		if err != nil {
			return "", err
		}
		targetPath += ";" + encodedProp
	}
	if debConfig != "" {
		targetPath += getDebianMatrixParams(debConfig)
	}
	return targetPath, nil
}

func prepareUploadData(targetPath, localPath, props string, uploadParams UploadParams, logMsgPrefix string) (os.FileInfo, string, string, error) {
	fileName, _ := fileutils.GetFileAndDirFromPath(targetPath)
	targetPath, err := addPropsToTargetPath(targetPath, props, uploadParams.GetDebian())
	if cliutils.CheckError(err) != nil {
		return nil, "", "", err
	}
	log.Info(logMsgPrefix + "Uploading artifact:", localPath)
	file, err := os.Open(localPath)
	defer file.Close()
	if cliutils.CheckError(err) != nil {
		return nil, "", "", err
	}
	fileInfo, err := file.Stat()
	if cliutils.CheckError(err) != nil {
		return nil, "", "", err
	}
	return fileInfo, targetPath, fileName, nil
}

// Uploads the file in the specified local path to the specified target path.
// Returns true if the file was successfully uploaded.
func (us *UploadService) uploadFile(localPath, targetPath, props string, uploadParams UploadParams, logMsgPrefix string) (utils.ArtifactsBuildInfo, bool, error) {
	fileInfo, targetPath, fileName, err := prepareUploadData(targetPath, localPath, props, uploadParams, logMsgPrefix)
	if err != nil {
		return utils.ArtifactsBuildInfo{}, false, err
	}
	file, err := os.Open(localPath)
	defer file.Close()
	if cliutils.CheckError(err) != nil {
		return utils.ArtifactsBuildInfo{}, false, err
	}
	var checksumDeployed bool = false
	var resp *http.Response
	var details *fileutils.FileDetails
	var body []byte
	httpClientsDetails := us.ArtDetails.CreateArtifactoryHttpClientDetails()
	fileStat, err := os.Lstat(localPath)
	if cliutils.CheckError(err) != nil {
		return utils.ArtifactsBuildInfo{}, false, err
	}
	if uploadParams.IsSymlink() && fileutils.IsFileSymlink(fileStat) {
		resp, details, body, err = us.uploadSymlink(targetPath, httpClientsDetails, uploadParams)
		if err != nil {
			return utils.ArtifactsBuildInfo{}, false, err
		}
	} else {
		resp, details, body, err = us.doUpload(file, localPath, targetPath, logMsgPrefix, httpClientsDetails, fileInfo, uploadParams)
	}
	if err != nil {
		return utils.ArtifactsBuildInfo{}, false, err
	}
	logUploadResponse(logMsgPrefix, resp, body, checksumDeployed, us.DryRun)
	artifact := createBuildArtifactItem(fileName, details)
	return artifact, us.DryRun || checksumDeployed || resp.StatusCode == 201 || resp.StatusCode == 200, nil
}

func (us *UploadService) uploadSymlink(targetPath string, httpClientsDetails httputils.HttpClientDetails, uploadParams UploadParams) (resp *http.Response, details *fileutils.FileDetails, body []byte, err error) {
	details = createSymlinkFileDetails()
	resp, body, err = utils.UploadFile(nil, targetPath, us.ArtDetails, details, httpClientsDetails, us.client)
	return
}

func (us *UploadService) doUpload(file *os.File, localPath, targetPath, logMsgPrefix string, httpClientsDetails httputils.HttpClientDetails, fileInfo os.FileInfo, uploadParams UploadParams) (*http.Response, *fileutils.FileDetails, []byte, error) {
	var details *fileutils.FileDetails
	var checksumDeployed bool
	var resp *http.Response
	var body []byte
	var err error
	addExplodeHeader(&httpClientsDetails, uploadParams.IsExplodeArchive())
	if fileInfo.Size() >= us.MinChecksumDeploy && !uploadParams.IsExplodeArchive() {
		resp, details, body, err = us.tryChecksumDeploy(localPath, targetPath, httpClientsDetails, us.client)
		if err != nil {
			return resp, details, body, err
		}
		checksumDeployed = !us.DryRun && (resp.StatusCode == 201 || resp.StatusCode == 200)
	}
	if !us.DryRun && !checksumDeployed {
		var body []byte
		resp, body, err = utils.UploadFile(file, targetPath, us.ArtDetails, details, httpClientsDetails, us.client)
		if err != nil {
			return resp, details, body, err
		}
		if resp.StatusCode != 201 && resp.StatusCode != 200 {
			log.Error(logMsgPrefix + "Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body))
		}
	}
	if !us.DryRun {
		var strChecksumDeployed string
		if checksumDeployed {
			strChecksumDeployed = " (Checksum deploy)"
		}
		log.Debug(logMsgPrefix, "Artifactory response:", resp.Status, strChecksumDeployed)
	}
	if details == nil {
		details, err = fileutils.GetFileDetails(localPath)
	}
	return resp, details, body, err
}

func logUploadResponse(logMsgPrefix string, resp *http.Response, body []byte, checksumDeployed, isDryRun bool) {
	if resp != nil && resp.StatusCode != 201 && resp.StatusCode != 200 {
		log.Error(logMsgPrefix + "Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body))
	}
	if !isDryRun {
		var strChecksumDeployed string
		if checksumDeployed {
			strChecksumDeployed = " (Checksum deploy)"
		} else {
			strChecksumDeployed = ""
		}
		log.Debug(logMsgPrefix, "Artifactory response:", resp.Status, strChecksumDeployed)
	}
}

// When handling symlink we want to simulate the creation of  empty file
func createSymlinkFileDetails() *fileutils.FileDetails {
	details := new(fileutils.FileDetails)
	details.Checksum.Md5, _ = fileutils.GetMd5(bytes.NewBuffer([]byte(fileutils.SYMLINK_FILE_CONTENT)))
	details.Checksum.Sha1, _ = fileutils.GetSha1(bytes.NewBuffer([]byte(fileutils.SYMLINK_FILE_CONTENT)))
	details.Size = int64(0)
	return details
}

func createBuildArtifactItem(fileName string, details *fileutils.FileDetails) utils.ArtifactsBuildInfo {
	return utils.ArtifactsBuildInfo{
		Name: fileName,
		BuildInfoCommon : &utils.BuildInfoCommon{
			Sha1: details.Checksum.Sha1,
			Md5: details.Checksum.Md5,
		},
	}
}

func addExplodeHeader(httpClientsDetails *httputils.HttpClientDetails, isExplode bool) {
	if isExplode {
		utils.AddHeader("X-Explode-Archive", "true", &httpClientsDetails.Headers)
	}
}

func (us *UploadService) tryChecksumDeploy(filePath, targetPath string,
	httpClientsDetails httputils.HttpClientDetails, client *helpers.JfrogHttpClient) (resp *http.Response, details *fileutils.FileDetails, body []byte, err error) {

	details, err = fileutils.GetFileDetails(filePath)
	if err != nil {
		return
	}
	headers := make(map[string]string)
	headers["X-Checksum-Deploy"] = "true"
	headers["X-Checksum-Sha1"] = details.Checksum.Sha1
	headers["X-Checksum-Md5"] = details.Checksum.Md5
	requestClientDetails := httpClientsDetails.Clone()
	cliutils.MergeMaps(headers, requestClientDetails.Headers)
	if us.DryRun {
		return
	}
	utils.AddAuthHeaders(headers, us.ArtDetails)
	cliutils.MergeMaps(headers, requestClientDetails.Headers)
	resp, body, err = client.SendPut(targetPath, nil, *requestClientDetails)
	return
}

func getDebianMatrixParams(debianPropsStr string) string {
	debProps := strings.Split(debianPropsStr, "/")
	return ";deb.distribution=" + debProps[0] +
		";deb.component=" + debProps[1] +
		";deb.architecture=" + debProps[2]
}

type UploadParamsImp struct {
	*utils.File
	Deb               string
	Symlink           bool
	ExplodeArchive    bool
}

func (up *UploadParamsImp) IsSymlink() bool {
	return up.Symlink
}

func (up *UploadParamsImp) IsExplodeArchive() bool {
	return up.ExplodeArchive
}

func (up *UploadParamsImp) GetDebian() string {
	return up.Deb
}

type UploadParams interface {
	utils.FileGetter
	IsSymlink() bool
	IsExplodeArchive() bool
	GetDebian() string
}

type UploadData struct {
	Artifact cliutils.Artifact
	Props    string
	IsDir    bool
}

type uploadDescriptor struct {
	IsFlat   bool
	IsRegexp bool
	IsDir    bool
	RootPath string
	Err      error
}

func (p *uploadDescriptor) CreateUploadDescriptor(isRegexp, isFlat, pattern string) {
	p.isRegexp(isRegexp)
	p.isFlat(isFlat)
	p.setRootPath(pattern)
	p.checkIfDir()
}

func (p *uploadDescriptor) isRegexp(isRegexpString string) {
	if p.Err == nil {
		p.IsRegexp, p.Err = cliutils.StringToBool(isRegexpString, false)
	}
}

func (p *uploadDescriptor) isFlat(isFlatString string) {
	if p.Err == nil {
		p.IsFlat, p.Err = cliutils.StringToBool(isFlatString, true)
	}
}

func (p *uploadDescriptor) setRootPath(pattern string) {
	if p.Err == nil {
		p.RootPath, p.Err = getRootPath(pattern, p.IsRegexp)
	}
}

func (p *uploadDescriptor) checkIfDir() {
	if p.Err == nil {
		p.IsDir, p.Err = fileutils.IsDir(p.RootPath)
	}
}

type uploadResult struct {
	UploadCount           []int
	TotalCount            []int
	BuildInfoArtifacts    [][]utils.ArtifactsBuildInfo
}

type artifactContext func(UploadData) parallel.TaskFunc

func (us *UploadService) createArtifactHandlerFunc(s *uploadResult, uploadParams UploadParams) artifactContext {
	return func(artifact UploadData) parallel.TaskFunc {
		return func(threadId int) (e error) {
			if artifact.IsDir {
				us.createFolderInArtifactory(artifact)
				return
			}
			var uploaded bool
			var target string
			var buildInfoArtifact utils.ArtifactsBuildInfo
			logMsgPrefix := cliutils.GetLogMsgPrefix(threadId, us.DryRun)
			target, e = utils.BuildArtifactoryUrl(us.ArtDetails.Url, artifact.Artifact.TargetPath, make(map[string]string))
			if e != nil {
				return
			}
			buildInfoArtifact, uploaded, e = us.uploadFile(artifact.Artifact.LocalPath, target, artifact.Props, uploadParams, logMsgPrefix)
			if e != nil {
				return
			}
			if uploaded {
				s.UploadCount[threadId]++
				s.BuildInfoArtifacts[threadId] = append(s.BuildInfoArtifacts[threadId], buildInfoArtifact)
			}
			s.TotalCount[threadId]++
			return
		}
	}
}

func (us *UploadService) createFolderInArtifactory(artifact UploadData) error {
	url, err := utils.BuildArtifactoryUrl(us.ArtDetails.Url, artifact.Artifact.TargetPath, make(map[string]string))
	url = cliutils.AddTrailingSlashIfNeeded(url)
	if err != nil {
		return err
	}
	content := make([]byte, 0)
	httpClientsDetails := us.ArtDetails.CreateArtifactoryHttpClientDetails()
	resp, body, err := us.client.SendPut(url, content, httpClientsDetails)
	if err != nil {
		log.Debug(resp)
		return err
	}
	logUploadResponse("Uploaded folder :", resp, body, false, us.DryRun)
	return err
}
