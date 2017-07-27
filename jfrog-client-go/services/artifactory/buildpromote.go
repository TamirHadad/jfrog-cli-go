package artifactory

import (
"encoding/json"
"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
"path"
"errors"
"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
)

type PromotionService struct {
	client *helpers.JfrogHttpClient
	ArtDetails      *auth.ArtifactoryAuthConfiguration
	DryRun          bool
}

func NewPromotionService(client *helpers.JfrogHttpClient) *PromotionService {
	return &PromotionService{client:client}
}

func (ps *PromotionService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return ps.ArtDetails
}

func (ps *PromotionService) SetArtifactoryDetails(rt *auth.ArtifactoryAuthConfiguration) {
	ps.ArtDetails = rt
}

func (ps *PromotionService) IsDryRun() bool {
	return ps.DryRun
}

func (ps *PromotionService) BuildPromote(promotionParams PromotionParams) error {
	message := "Promoting build..."
	if ps.DryRun == true {
		message = "[Dry run] " + message
	}
	log.Info(message)

	promoteUrl := ps.ArtDetails.Url
	restApi := path.Join("api/build/promote/", promotionParams.GetBuildName(), promotionParams.GetBuildNumber())
	requestFullUrl, err := utils.BuildArtifactoryUrl(promoteUrl, restApi, make(map[string]string))
	if err != nil {
		return err
	}

	data := BuildPromotionBody{
		Status:                 promotionParams.GetStatus(),
		Comment :               promotionParams.GetComment(),
		Copy:                   promotionParams.IsCopy(),
		IncludeDependencies:    promotionParams.IsIncludeDependencies(),
		SourceRepo:             promotionParams.GetSourceRepo(),
		TargetRepo:             promotionParams.GetTargetRepo(),
		DryRun:                 ps.IsDryRun()}
	requestContent, err := json.Marshal(data)
	if err != nil {
		return cliutils.CheckError(errors.New("Failed to execute request. " + cliutils.GetDocumentationMessage()))
	}

	httpClientsDetails := ps.ArtDetails.CreateArtifactoryHttpClientDetails()
	utils.SetContentType("application/vnd.org.jfrog.artifactory.build.PromotionRequest+json", &httpClientsDetails.Headers)

	resp, body, err := ps.client.SendPost(requestFullUrl, requestContent, httpClientsDetails)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return cliutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body)))
	}

	log.Debug("Artifactory response:", resp.Status)
	log.Info("Promoted build", promotionParams.GetBuildName() , "#" + promotionParams.GetBuildNumber(), "to:", promotionParams.GetTargetRepo(), "repository.")
	return nil
}


type BuildPromotionBody struct {
	Comment             string `json:"comment,omitempty"`
	SourceRepo          string `json:"sourceRepo,omitempty"`
	TargetRepo          string `json:"targetRepo,omitempty"`
	Status              string `json:"status,omitempty"`
	IncludeDependencies bool   `json:"dependencies,omitempty"`
	Copy                bool   `json:"copy,omitempty"`
	DryRun              bool   `json:"dryRun,omitempty"`
}

type PromotionParams interface {
	GetBuildName() string
	GetBuildNumber() string
	GetTargetRepo() string
	GetStatus() string
	GetComment() string
	IsCopy() bool
	IsIncludeDependencies() bool
	GetSourceRepo() string
}

type PromotionParamsImpl struct {
	BuildName string
	BuildNumber string
	TargetRepo string
	Status string
	Comment string
	Copy bool
	IncludeDependencies bool
	SourceRepo string
}

func (bp *PromotionParamsImpl) GetBuildName() string {
	return bp.BuildName
}

func (bp *PromotionParamsImpl) GetBuildNumber() string {
	return bp.BuildNumber
}

func (bp *PromotionParamsImpl) GetTargetRepo() string {
	return bp.TargetRepo
}

func (bp *PromotionParamsImpl) GetStatus() string {
	return bp.Status
}

func (bp *PromotionParamsImpl) GetComment() string {
	return bp.Comment
}

func (bp *PromotionParamsImpl) IsCopy() bool {
	return bp.Copy
}

func (bp *PromotionParamsImpl) IsIncludeDependencies() bool {
	return bp.IncludeDependencies
}

func (bp *PromotionParamsImpl) GetSourceRepo() string {
	return bp.SourceRepo
}




