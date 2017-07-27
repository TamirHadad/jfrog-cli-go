package commands

import (
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"os"
	"strconv"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory"
	clientutils "github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"time"
	"strings"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services"
)

// Uploads the artifacts in the specified local path pattern to the specified target path.
// Returns the total number of artifacts successfully uploaded.
func Upload(uploadSpec *utils.SpecFiles, flags *UploadConfiguration) (totalUploaded, totalFailed int, err error) {
	certPath, err := utils.GetJfrogSecurityDir()
	if err != nil {
		return 0, 0, err
	}
	minChecksumDeploySize, err := getMinChecksumDeploySize()
	if err != nil {
		return 0, 0, err
	}

	servicesConfig, err := createUploadServiceConfig(flags.ArtDetails, flags, certPath, minChecksumDeploySize)
	if err != nil {
		return 0, 0, err
	}
	servicesManager, err := services.NewArtifactoryService(servicesConfig)
	if err != nil {
		return 0, 0, err
	}
	isCollectBuildInfo := len(flags.BuildName) > 0 && len(flags.BuildNumber) > 0
	if isCollectBuildInfo && !flags.DryRun {
		if err := utils.SaveBuildGeneralDetails(flags.BuildName, flags.BuildNumber); err != nil {
			return 0, 0, err
		}
		for i := 0; i < len(uploadSpec.Files); i++ {
			addBuildProps(&uploadSpec.Get(i).Props, flags.BuildName, flags.BuildNumber)
		}
	}

	uploadParamImp := createBaseUploadParams(flags)
	var buildArtifacts []clientutils.ArtifactsBuildInfo
	for i := 0; i < len(uploadSpec.Files); i++ {
		uploadParamImp.File = uploadSpec.Get(i)
		artifacts, uploaded, failed, err := servicesManager.UploadFiles(uploadParamImp)
		if err != nil {
			return 0, 0, err
		}
		buildArtifacts = append(buildArtifacts, artifacts...)
		totalFailed += failed
		totalUploaded += uploaded
	}
	if err != nil {
		return 0, 0, err
	}
	if totalFailed > 0 {
		return
	}
	if isCollectBuildInfo && !flags.DryRun {
		populateFunc := func(tempWrapper *utils.ArtifactBuildInfoWrapper) {
			tempWrapper.Artifacts = buildArtifacts
		}
		err = utils.SavePartialBuildInfo(flags.BuildName, flags.BuildNumber, populateFunc)
	}
	return
}

func createUploadServiceConfig(artDetails *config.ArtifactoryDetails, flags *UploadConfiguration, certPath string, minChecksumDeploySize int64) (services.ArtifactoryConfig, error) {
	servicesConfig, err := new(services.ArtifactoryServicesConfigBuilder).
		SetArtDetails(artDetails.CreateArtAuthConfig()).
		SetDryRun(flags.DryRun).
		SetCertifactesPath(certPath).
		SetMinChecksumDeploy(minChecksumDeploySize).
		SetNumOfThreadPerOperation(flags.Threads).
		Build()
	return servicesConfig, err
}

func createBaseUploadParams(flags *UploadConfiguration) (*artifactory.UploadParamsImp) {
	uploadParamImp := &artifactory.UploadParamsImp{}
	uploadParamImp.Deb = flags.Deb
	uploadParamImp.Symlink = flags.Symlink
	uploadParamImp.ExplodeArchive = flags.ExplodeArchive
	return uploadParamImp
}

func getMinChecksumDeploySize() (int64, error) {
	minChecksumDeploySize := os.Getenv("JFROG_CLI_MIN_CHECKSUM_DEPLOY_SIZE_KB")
	if minChecksumDeploySize == "" {
		return 10240, nil
	}
	minSize, err := strconv.ParseInt(minChecksumDeploySize, 10, 64)
	err = cliutils.CheckError(err)
	if err != nil {
		return 0, err
	}
	return minSize * 1000, nil
}

func addBuildProps(props *string, buildName, buildNumber string) (err error) {
	if buildName == "" || buildNumber == "" {
		return
	}
	buildProps := "build.name=" + buildName
	buildProps += ";build.number=" + buildNumber
	buildGeneralDetails, err := utils.ReadBuildInfoGeneralDetails(buildName, buildNumber)
	if err != nil {
		return
	}
	buildProps += ";build.timestamp=" + strconv.FormatInt(buildGeneralDetails.Timestamp.UnixNano() / int64(time.Millisecond), 10)
	*props = addProps(*props, buildProps)
	return
}

func addProps(oldProps, additionalProps string) string {
	if len(oldProps) > 0 && !strings.HasSuffix(oldProps, ";")  && len(additionalProps) > 0 {
		oldProps += ";"
	}
	return oldProps + additionalProps
}

type UploadConfiguration struct {
	Deb                   string
	Threads               int
	MinChecksumDeploySize int64
	BuildName             string
	BuildNumber           string
	DryRun                bool
	Symlink               bool
	ExplodeArchive        bool
    ArtDetails            *config.ArtifactoryDetails
}