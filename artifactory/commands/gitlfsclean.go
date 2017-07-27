package commands

import (
	clientutils "github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
)

func PrepareGitLfsClean(flags *GitLfsCleanConfiguration) ([]clientutils.AqlSearchResultItem, error) {
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return nil, err
	}
	return servicesManager.GetRedundantGitLfsFiles(flags)
}

func DeleteLfsFilesFromArtifactory(files []clientutils.AqlSearchResultItem, flags *GitLfsCleanConfiguration) error {
	log.Info("Deleting", len(files), "files from", flags.Repo, "...")
	servicesManager, err := utils.CreateDefaultServiceManager(flags.ArtDetails, flags.DryRun)
	if err != nil {
		return err
	}
	err = servicesManager.DeleteFiles(files)
	if err != nil {
		return cliutils.CheckError(err)
	}
	return nil
}

type GitLfsCleanConfiguration struct {
	*artifactory.GitLfsCleanParamsImpl
	ArtDetails *config.ArtifactoryDetails
	Quiet bool
	DryRun bool
}
