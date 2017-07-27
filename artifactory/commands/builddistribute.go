package commands

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
)

func BuildDistribute(flags *BuildDistributionConfiguration) error {
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return err
	}
	return servicesManager.BuildDistribute(flags.BuildDistributionParamsImpl)
}

type BuildDistributionConfiguration struct {
	*artifactory.BuildDistributionParamsImpl
	ArtDetails *config.ArtifactoryDetails
	DryRun bool
}
