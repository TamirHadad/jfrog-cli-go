package commands

import (
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	clientutils "github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"strconv"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
)

func Download(downloadSpec *utils.SpecFiles, flags *DownloadConfiguration) error {
	servicesManager, err := createDownloadServiceManager(flags.ArtDetails, flags)
	if err != nil {
		return err
	}
	isCollectBuildInfo := len(flags.BuildName) > 0 && len(flags.BuildNumber) > 0
	if isCollectBuildInfo && !flags.DryRun {
		if err = utils.SaveBuildGeneralDetails(flags.BuildName, flags.BuildNumber); err != nil {
			return err
		}
	}
	if !flags.DryRun {
		err = fileutils.CreateTempDirPath()
		if err != nil {
			return err
		}
		defer fileutils.RemoveTempDir()
	}
	var buildDependencies []clientutils.DependenciesBuildInfo
	for i := 0; i < len(downloadSpec.Files); i++ {
		currentSpec := downloadSpec.Get(i)
		currentBuildDependencies, err := servicesManager.DownloadFiles(&artifactory.DownloadParamsImpl{File: currentSpec, ValidateSymlink: flags.ValidateSymlink, Symlink: flags.Symlink})
		if err != nil {
			return err
		}
		buildDependencies = append(buildDependencies, currentBuildDependencies...)
	}
	log.Info("Downloaded", strconv.Itoa(len(buildDependencies)), "artifacts.")
	if isCollectBuildInfo && !flags.DryRun {
		populateFunc := func(tempWrapper *utils.ArtifactBuildInfoWrapper) {
			tempWrapper.Dependencies = buildDependencies
		}
		err = utils.SavePartialBuildInfo(flags.BuildName, flags.BuildNumber, populateFunc)
	}
	return err
}

type DownloadConfiguration struct {
	Threads         int
	SplitCount      int
	MinSplitSize    int64
	BuildName       string
	BuildNumber     string
	DryRun          bool
	Symlink         bool
	ValidateSymlink bool
	ArtDetails      *config.ArtifactoryDetails
}

func createDownloadServiceManager(artDetails *config.ArtifactoryDetails, flags *DownloadConfiguration) (*services.ArtifactoryServicesManager, error) {
	certPath, err := utils.GetJfrogSecurityDir()
	if err != nil {
		return nil, err
	}
	serviceConfig, err := (&services.ArtifactoryServicesConfigBuilder{}).
		SetArtDetails(artDetails.CreateArtAuthConfig()).
		SetDryRun(flags.DryRun).
		SetCertifactesPath(certPath).
		SetSplitCount(flags.SplitCount).
		SetMinSplitSize(flags.MinSplitSize).
		SetNumOfThreadPerOperation(flags.Threads).
		Build()
	if err != nil {
		return nil, err
	}
	return services.NewArtifactoryService(serviceConfig)
}