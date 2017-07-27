package artifactory

import (
"encoding/json"
"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
"path"
"errors"
"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
"strings"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
)

type DistributionService struct {
	client     *helpers.JfrogHttpClient
	ArtDetails *auth.ArtifactoryAuthConfiguration
	DryRun     bool
}

func (ds *DistributionService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return ds.ArtDetails
}

func (ds *DistributionService) SetArtifactoryDetails(artDetails *auth.ArtifactoryAuthConfiguration) {
	ds.ArtDetails = artDetails
}

func (ds *DistributionService) IsDryRun() bool {
	return ds.DryRun
}

func NewDistributionService(client *helpers.JfrogHttpClient) *DistributionService {
	return &DistributionService{client:client}
}

func (ds *DistributionService) BuildDistribute(params BuildDistributionParams) error {
	dryRun := ""
	if ds.DryRun == true {
		dryRun = "[Dry run] "
	}
	message := "Distributing build..."
	log.Info(dryRun + message)

	distributeUrl := ds.ArtDetails.Url
	restApi := path.Join("api/build/distribute/", params.GetBuildName(), params.GetBuildNumber())
	requestFullUrl, err := utils.BuildArtifactoryUrl(distributeUrl, restApi, make(map[string]string))
	if err != nil {
		return err
	}

	data := BuildDistributionBody{
		SourceRepos:           strings.Split(params.GetSourceRepos(), ","),
		TargetRepo:            params.GetTargetRepo(),
		Publish:               params.IsPublish(),
		OverrideExistingFiles: params.IsOverrideExistingFiles(),
		GpgPassphrase:         params.GetGpgPassphrase(),
		Async:                 params.IsAsync(),
		DryRun:                ds.IsDryRun()}
	requestContent, err := json.Marshal(data)
	if err != nil {
		return cliutils.CheckError(errors.New("Failed to execute request. " + cliutils.GetDocumentationMessage()))
	}

	httpClientsDetails := ds.GetArtifactoryDetails().CreateArtifactoryHttpClientDetails()
	utils.SetContentType("application/json", &httpClientsDetails.Headers)

	resp, body, err := ds.client.SendPost(requestFullUrl, requestContent, httpClientsDetails)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return cliutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body)))
	}

	log.Debug("Artifactory response:", resp.Status)
	if params.IsAsync() && !ds.IsDryRun() {
		log.Info("Asynchronously distributed build", params.GetBuildName(), "#" + params.GetBuildNumber(), "to:", params.GetTargetRepo(), "repository, logs are avalable in Artifactory.")
		return nil
	}

	log.Info(dryRun + "Distributed build", params.GetBuildName(), "#" + params.GetBuildNumber(), "to:", params.GetTargetRepo(), "repository.")
	return nil
}

type BuildDistributionParams interface {
	GetSourceRepos() string
	GetTargetRepo() string
	GetGpgPassphrase() string
	IsAsync() bool
	IsPublish() bool
	IsOverrideExistingFiles() bool
	GetBuildName() string
	GetBuildNumber() string
}

type BuildDistributionParamsImpl struct {
	SourceRepos           string
	TargetRepo            string
	GpgPassphrase         string
	Publish               bool
	OverrideExistingFiles bool
	Async                 bool
	BuildName             string
	BuildNumber           string
}

func (bd *BuildDistributionParamsImpl) GetSourceRepos() string {
	return bd.SourceRepos
}

func (bd *BuildDistributionParamsImpl) GetTargetRepo() string {
	return bd.SourceRepos
}

func (bd *BuildDistributionParamsImpl) GetGpgPassphrase() string {
	return bd.GpgPassphrase
}

func (bd *BuildDistributionParamsImpl) IsAsync() bool {
	return bd.Async
}

func (bd *BuildDistributionParamsImpl) IsPublish() bool {
	return bd.Publish
}

func (bd *BuildDistributionParamsImpl) IsOverrideExistingFiles() bool  {
	return bd.OverrideExistingFiles
}

func (bd *BuildDistributionParamsImpl) GetBuildName() string  {
	return bd.BuildName
}

func (bd *BuildDistributionParamsImpl) GetBuildNumber() string  {
	return bd.BuildNumber
}


type BuildDistributionBody struct {
	SourceRepos           []string  `json:"sourceRepos,omitempty"`
	TargetRepo            string    `json:"targetRepo,omitempty"`
	GpgPassphrase         string    `json:"gpgPassphrase,omitempty"`
	Publish               bool      `json:"publish"`
	OverrideExistingFiles bool      `json:"overrideExistingFiles,omitempty"`
	Async                 bool      `json:"async,omitempty"`
	DryRun                bool      `json:"dryRun,omitempty"`
}
