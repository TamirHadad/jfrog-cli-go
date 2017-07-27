package commands

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	clientutils "github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
)

func Delete(deleteSpec *utils.SpecFiles, flags *DeleteConfiguration) (err error) {
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return err
	}
	var resultItems []clientutils.AqlSearchResultItem
	for i := 0; i < len(deleteSpec.Files); i++ {
		currentSpec := deleteSpec.Get(i)
		currentResultItems, err := servicesManager.GetPathsToDelete(&artifactory.DeleteParamsImpl{File: currentSpec})
		if err != nil {
			return err
		}
		resultItems = append(resultItems, currentResultItems...)
	}
	if err = servicesManager.DeleteFiles(resultItems); err != nil {
		return
	}
	log.Info("Deleted", len(resultItems), "items.")
	return
}

func DeleteFiles(resultItems []clientutils.AqlSearchResultItem, flags *DeleteConfiguration) error {
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return err
	}
	return servicesManager.DeleteFiles(resultItems)
}

func GetPathsToDelete(deleteSpec *utils.SpecFiles, flags *DeleteConfiguration) ([]clientutils.AqlSearchResultItem, error) {
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return nil, err
	}
	var resultItems []clientutils.AqlSearchResultItem
	for i := 0; i < len(deleteSpec.Files); i++ {
		currentSpec := deleteSpec.Get(i)
		currentResultItems, err := servicesManager.GetPathsToDelete(&artifactory.DeleteParamsImpl{File: currentSpec})
		if err != nil {
			return nil, err
		}
		resultItems = append(resultItems, currentResultItems...)
	}
	return resultItems, nil
}

type DeleteConfiguration struct {
	ArtDetails *config.ArtifactoryDetails
	DryRun     bool
}
