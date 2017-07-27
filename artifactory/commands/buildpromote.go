package commands

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
)

func BuildPromote(flags *BuildPromotionConfiguration) error {
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return err
	}
	return servicesManager.BuildPromote(flags.PromotionParamsImpl)
}

type BuildPromotionConfiguration struct {
	*artifactory.PromotionParamsImpl
	ArtDetails *config.ArtifactoryDetails
	DryRun bool
}
