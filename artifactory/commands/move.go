package commands

import (
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
)

// Moves the artifacts using the specified move pattern.
func Move(moveSpec *utils.SpecFiles, artDetails *config.ArtifactoryDetails) error {
	servicesManager, err := utils.CreateDefaultServiceManager(artDetails, false)
	if err != nil {
		return err
	}
	for i := 0; i < len(moveSpec.Files); i++ {
		currentSpec := moveSpec.Get(i)
		err = servicesManager.Move(&artifactory.MoveCopyParamsImpl{File:currentSpec})
		if err != nil {
			return err
		}
	}
	return nil
}