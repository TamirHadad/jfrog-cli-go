package artifactory

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"errors"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
)

type DeleteService struct {
	client *helpers.JfrogHttpClient
	ArtDetails      *auth.ArtifactoryAuthConfiguration
	DryRun          bool
}

func (ds *DeleteService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return ds.ArtDetails
}

func (ds *DeleteService) SetArtifactoryDetails(rt *auth.ArtifactoryAuthConfiguration) {
	ds.ArtDetails = rt
}

func (ds *DeleteService) IsDryRun() bool {
	return ds.DryRun
}

func NewDeleteService(client *helpers.JfrogHttpClient) *DeleteService {
	return &DeleteService{client:client}
}

func (ds *DeleteService) GetPathsToDelete(deleteParams DeleteParams) (resultItems []utils.AqlSearchResultItem, err error) {
	log.Info("Searching artifacts...")
	// Search paths using AQL.
	if deleteParams.GetSpecType() == utils.AQL {
		if resultItemsTemp, e := utils.AqlSearchBySpec(deleteParams.GetFile(), ds, ds.client); e == nil {
			resultItems = append(resultItems, resultItemsTemp...)
		} else {
			err = e
			return
		}
	} else {

		deleteParams.SetIncludeDirs("true")
		tempResultItems, e := utils.AqlSearchDefaultReturnFields(deleteParams.GetFile(), ds, ds.client)
		if e != nil {
			err = e
			return
		}
		paths := utils.ReduceDirResult(tempResultItems, utils.FilterTopChainResults)
		resultItems = append(resultItems, paths...)
	}
	utils.LogSearchResults(len(resultItems))
	return
}

func (ds *DeleteService) DeleteFiles(resultItems []utils.AqlSearchResultItem, conf utils.CommonConf) error {
	for _, v := range resultItems {
		fileUrl, err := utils.BuildArtifactoryUrl(conf.GetArtifactoryDetails().Url, v.GetFullUrl(), make(map[string]string))
		if err != nil {
			return err
		}
		if conf.IsDryRun() {
			log.Info("[Dry run] Deleting:", v.GetFullUrl())
			continue
		}

		log.Info("Deleting:", v.GetFullUrl())
		httpClientsDetails := conf.GetArtifactoryDetails().CreateArtifactoryHttpClientDetails()
		resp, body, err := ds.client.SendDelete(fileUrl, nil, httpClientsDetails)
		if err != nil {
			return err
		}
		if resp.StatusCode != 204 {
			return cliutils.CheckError(errors.New("Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body)))
		}

		log.Debug("Artifactory response:", resp.Status)
	}
	return nil
}

type DeleteConfiguration struct {
	ArtDetails *auth.ArtifactoryAuthConfiguration
	DryRun     bool
}

func (conf *DeleteConfiguration) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return conf.ArtDetails
}

func (conf *DeleteConfiguration) SetArtifactoryDetails(art *auth.ArtifactoryAuthConfiguration) {
	conf.ArtDetails = art
}

func (conf *DeleteConfiguration) IsDryRun() bool {
	return conf.DryRun
}

type DeleteParams interface {
	utils.FileGetter
	GetFile() *utils.File
	SetIncludeDirs(includeDirs string)
}

type DeleteParamsImpl struct {
	*utils.File
}

func (ds *DeleteParamsImpl) GetFile() *utils.File {
	return ds.File
}

func (ds *DeleteParamsImpl) SetIncludeDirs(includeDirs string) {
	ds.IncludeDirs = includeDirs
}
