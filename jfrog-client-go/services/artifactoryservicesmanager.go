package services

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	"net/http"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth/cert"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
)

func (builder *ArtifactoryServicesConfigBuilder) SetUrl(url string) *ArtifactoryServicesConfigBuilder {
	builder.Url = url
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetArtDetails(artDetails *auth.ArtifactoryAuthConfiguration) *ArtifactoryServicesConfigBuilder {
	builder.ArtifactoryAuthConfiguration = artDetails
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetPassword(password string) *ArtifactoryServicesConfigBuilder {
	builder.Password = password
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetApiKey(apiKey string) *ArtifactoryServicesConfigBuilder {
	builder.ApiKey = apiKey
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetSshKeysPath(sshKeysPath string) *ArtifactoryServicesConfigBuilder {
	builder.SshKeysPath = sshKeysPath
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetCertifactesPath(certifactesPath string) *ArtifactoryServicesConfigBuilder {
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetNumOfThreadPerOperation(threads int) *ArtifactoryServicesConfigBuilder {
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetMinSplitSize(splitSize int64) *ArtifactoryServicesConfigBuilder {
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetSplitCount(splitCount int) *ArtifactoryServicesConfigBuilder {
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetMinChecksumDeploy(minChecksumDeploy int64) *ArtifactoryServicesConfigBuilder {
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) SetDryRun(dryRun bool) *ArtifactoryServicesConfigBuilder {
	builder.isDryRun = dryRun
	return builder
}

func (builder *ArtifactoryServicesConfigBuilder) Build() (ArtifactoryConfig, error) {
	c := &artifactoryServicesConfig{}
	if builder.SshKeysPath != "" {
		header, err := builder.ArtifactoryAuthConfiguration.SshAuthentication()
		if err != nil {
			return nil, err
		}
		c.SshAuthHeaders = header
	}
	c.ArtifactoryAuthConfiguration = builder.ArtifactoryAuthConfiguration

	if builder.threads == 0 {
		c.threads = 3
	} else {
		c.threads = builder.threads
	}

	if builder.minChecksumDeploy == 0 {
		c.minChecksumDeploy = 10240
	} else {
		c.minChecksumDeploy = builder.minChecksumDeploy
	}

	c.minSplitSize = builder.minSplitSize
	c.splitCount = builder.splitCount
	return c, nil
}

type ArtifactoryServicesConfigBuilder struct {
	*auth.ArtifactoryAuthConfiguration
	certifactesPath   string
	threads           int
	minSplitSize      int64
	splitCount        int
	minChecksumDeploy int64
	isDryRun          bool
}

type artifactoryServicesConfig struct {
	*auth.ArtifactoryAuthConfiguration
	certifactesPath   string
	dryRun            bool
	threads           int
	minSplitSize      int64
	splitCount        int
	minChecksumDeploy int64
	isDryRun          bool
}

type ArtifactoryConfig interface {
	GetUrl() string
	GetPassword() string
	GetApiKey() string
	GetSshKeyPath() string
	GetCertifactesPath() string
	GetNumOfThreadPerOperation() int
	GetMinSplitSize() int64
	GetSplitCount() int
	GetMinChecksumDeploy() int64
	IsDryRun() bool
	GetArtDetails() *auth.ArtifactoryAuthConfiguration
}

func (config *artifactoryServicesConfig) GetUrl() string {
	return config.Url
}

func (config *artifactoryServicesConfig) IsDryRun() bool {
	return config.isDryRun
}

func (config *artifactoryServicesConfig) GetPassword() string {
	return config.Password
}

func (config *artifactoryServicesConfig) GetApiKey() string {
	return config.ApiKey
}

func (config *artifactoryServicesConfig) GetSshKeyPath() string {
	return config.SshKeysPath
}

func (config *artifactoryServicesConfig) GetCertifactesPath() string {
	return config.certifactesPath
}

func (config *artifactoryServicesConfig) GetNumOfThreadPerOperation() int {
	return config.threads
}

func (config *artifactoryServicesConfig) GetMinSplitSize() int64 {
	return config.minSplitSize
}

func (config *artifactoryServicesConfig) GetSplitCount() int {
	return config.splitCount
}
func (config *artifactoryServicesConfig) GetMinChecksumDeploy() int64 {
	return config.minChecksumDeploy
}

func (config *artifactoryServicesConfig) GetArtDetails() *auth.ArtifactoryAuthConfiguration {
	return config.ArtifactoryAuthConfiguration
}

type ArtifactoryServicesManager struct {
	artClient *helpers.JfrogHttpClient
	config    ArtifactoryConfig
}

func NewArtifactoryService(config ArtifactoryConfig) (*ArtifactoryServicesManager, error) {
	var err error
	manager := &ArtifactoryServicesManager{config: config}
	if config.GetCertifactesPath() == "" {
		manager.artClient = helpers.NewDefaultJforgHttpClient()
	} else {
		transport, err := cert.GetTransportWithLoadedCert(config.GetCertifactesPath())
		if err != nil {
			return nil, err
		}
		manager.artClient = helpers.NewJforgHttpClient(&http.Client{Transport: transport})
	}
	return manager, err
}

func (sm *ArtifactoryServicesManager) BuildDistribute(params artifactory.BuildDistributionParams) error {
	distributionService := artifactory.NewDistributionService(sm.artClient)
	distributionService.DryRun = sm.config.IsDryRun()
	distributionService.ArtDetails = sm.config.GetArtDetails()
	return distributionService.BuildDistribute(params)
}

func (sm *ArtifactoryServicesManager) BuildPromote(params artifactory.PromotionParams) error {
	promotionService := artifactory.NewPromotionService(sm.artClient)
	promotionService.DryRun = sm.config.IsDryRun()
	promotionService.ArtDetails = sm.config.GetArtDetails()
	return promotionService.BuildPromote(params)
}

func (sm *ArtifactoryServicesManager) GetPathsToDelete(params artifactory.DeleteParams) ([]utils.AqlSearchResultItem, error) {
	deleteService := artifactory.NewDeleteService(sm.artClient)
	deleteService.DryRun = sm.config.IsDryRun()
	deleteService.ArtDetails = sm.config.GetArtDetails()
	return deleteService.GetPathsToDelete(params)
}

func (sm *ArtifactoryServicesManager) DeleteFiles(resultItems []utils.AqlSearchResultItem) error {
	deleteService := artifactory.NewDeleteService(sm.artClient)
	deleteService.DryRun = sm.config.IsDryRun()
	deleteService.ArtDetails = sm.config.GetArtDetails()
	return deleteService.DeleteFiles(resultItems, deleteService)
}

func (sm *ArtifactoryServicesManager) DownloadFiles(params artifactory.DownloadParams) ([]utils.DependenciesBuildInfo, error) {
	downloadService := artifactory.NewDownloadService(sm.artClient)
	downloadService.DryRun = sm.config.IsDryRun()
	downloadService.ArtDetails = sm.config.GetArtDetails()
	downloadService.Threads = sm.config.GetNumOfThreadPerOperation()
	downloadService.SplitCount = sm.config.GetSplitCount()
	downloadService.MinSplitSize = sm.config.GetMinSplitSize()
	return downloadService.DownloadFiles(params)
}

func (sm *ArtifactoryServicesManager) GetRedundantGitLfsFiles(params artifactory.GitLfsCleanParams) ([]utils.AqlSearchResultItem, error) {
	gitLfsCleanService := artifactory.NewGitLfsCleanService(sm.artClient)
	gitLfsCleanService.DryRun = sm.config.IsDryRun()
	gitLfsCleanService.ArtDetails = sm.config.GetArtDetails()
	return gitLfsCleanService.GetRedundantGitLfsFiles(params)
}

func (sm *ArtifactoryServicesManager) Search(params utils.SearchParams) ([]utils.AqlSearchResultItem, error) {
	searchService := artifactory.NewSearchService(sm.artClient)
	searchService.ArtDetails = sm.config.GetArtDetails()
	return searchService.Search(params, searchService)
}

func (sm *ArtifactoryServicesManager) SetProps(params artifactory.SetPropsParams) error {
	setPropsService := artifactory.NewSetPropsService(sm.artClient)
	setPropsService.ArtDetails = sm.config.GetArtDetails()
	return setPropsService.SetProps(params)
}

func (sm *ArtifactoryServicesManager) UploadFiles(params artifactory.UploadParams) ([]utils.ArtifactsBuildInfo, int, int, error) {
	uploadService := artifactory.NewUploadService(sm.artClient)
	sm.setCommonServiceConfig(uploadService)
	uploadService.MinChecksumDeploy = sm.config.GetMinChecksumDeploy()
	return uploadService.UploadFiles(params)
}

func (sm *ArtifactoryServicesManager) Copy(params artifactory.MoveCopyParams) error {
	copyService := artifactory.NewMoveCopyService(sm.artClient, artifactory.COPY)
	copyService.ArtDetails = sm.config.GetArtDetails()
	return copyService.MoveCopyServiceMoveFilesWrapper(params)
}

func (sm *ArtifactoryServicesManager) Move(params artifactory.MoveCopyParams) error {
	moveService := artifactory.NewMoveCopyService(sm.artClient, artifactory.MOVE)
	moveService.ArtDetails = sm.config.GetArtDetails()
	return moveService.MoveCopyServiceMoveFilesWrapper(params)
}

func (sm *ArtifactoryServicesManager) setCommonServiceConfig(commonConfig CommonServicesSetter) {
	commonConfig.SetThread(sm.config.GetNumOfThreadPerOperation())
	commonConfig.SetArtDetails(sm.config.GetArtDetails())
	commonConfig.SetDryRun(sm.config.IsDryRun())
}

type CommonServicesSetter interface {
	SetThread(threads int)
	SetArtDetails(artDetails *auth.ArtifactoryAuthConfiguration)
	SetDryRun(isDryRun bool)
}
