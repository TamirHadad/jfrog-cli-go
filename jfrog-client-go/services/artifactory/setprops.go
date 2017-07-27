package artifactory

import (
"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
"github.com/jfrogdev/jfrog-cli-go/errors/httperrors"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
)

type SetPropsService struct {
	client *helpers.JfrogHttpClient
	ArtDetails *auth.ArtifactoryAuthConfiguration
}

func (sp *SetPropsService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return sp.ArtDetails
}

func (sp *SetPropsService) SetArtifactoryDetails(rt *auth.ArtifactoryAuthConfiguration) {
	sp.ArtDetails = rt
}

func (sp *SetPropsService) IsDryRun() bool {
	return false
}

func NewSetPropsService(client *helpers.JfrogHttpClient) *SetPropsService {
	return &SetPropsService{client:client}
}

func (sp *SetPropsService) SetProps(setPropsParams SetPropsParams) error {
	updatePropertiesBaseUrl := sp.GetArtifactoryDetails().Url + "api/storage"
	log.Info("Setting properties...")
	encodedParam, err := utils.EncodeParams(setPropsParams.GetProps())
	if err != nil {
		return err
	}
	for _, item := range setPropsParams.GetItems() {
		log.Info("Setting properties to:", item.GetFullUrl())
		httpClientsDetails := sp.GetArtifactoryDetails().CreateArtifactoryHttpClientDetails()
		setPropertiesUrl := updatePropertiesBaseUrl + "/" + item.GetFullUrl() + "?properties=" + encodedParam
		log.Debug("Sending set properties request:", setPropertiesUrl)
		resp, body, err := sp.client.SendPut(setPropertiesUrl, nil, httpClientsDetails)
		if err != nil {
			return err
		}
		if err = httperrors.CheckResponseStatus(resp, body, 204); err != nil {
			return err
		}
	}

	log.Info("Done setting properties.")
	return err
}

type SetPropsParams interface {
	GetItems() []utils.AqlSearchResultItem
	GetProps() string
}

type SetPropsParamsImpl struct {
	Items []utils.AqlSearchResultItem
	Props string
}

func (sp *SetPropsParamsImpl) GetItems() []utils.AqlSearchResultItem {
	return sp.Items
}

func (sp *SetPropsParamsImpl) GetProps() string {
	return sp.Props
}