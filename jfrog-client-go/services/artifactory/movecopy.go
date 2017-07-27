package artifactory

import (
	"strings"
	"strconv"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/httputils"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"errors"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
)

const (
	MOVE MoveType = "move"
	COPY MoveType = "copy"
)

type MoveCopyService struct {
	moveType MoveType
	client *helpers.JfrogHttpClient
	ArtDetails *auth.ArtifactoryAuthConfiguration
}

func (mc *MoveCopyService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return mc.ArtDetails
}

func (mc *MoveCopyService) SetArtifactoryDetails(rt *auth.ArtifactoryAuthConfiguration) {
	mc.ArtDetails = rt
}

func (mc *MoveCopyService) IsDryRun() bool {
	return false
}

func NewMoveCopyService(client *helpers.JfrogHttpClient, moveType MoveType) *MoveCopyService {
	return &MoveCopyService{moveType:moveType, client:client}
}

func (mc *MoveCopyService) MoveCopyServiceMoveFilesWrapper(moveSpec MoveCopyParams) (err error) {
	var successCount int
	var failedCount int

	var successPartial, failedPartial int
	switch moveSpec.GetSpecType() {
	case utils.WILDCARD, utils.SIMPLE:
		successPartial, failedPartial, err = mc.moveWildcard(moveSpec.GetFile())
	case utils.AQL:
		successPartial, failedPartial, err = mc.moveAql(moveSpec.GetFile())
	}
	successCount += successPartial
	failedCount += failedPartial
	if err != nil {
		return
	}

	log.Info(moveMsgs[mc.moveType].MovedMsg, strconv.Itoa(successCount), "artifacts.")
	if failedCount > 0 {
		err = cliutils.CheckError(errors.New("Failed " + moveMsgs[mc.moveType].MovingMsg + " " + strconv.Itoa(failedCount) + " artifacts."))
	}

	return
}

func (mc *MoveCopyService) moveAql(fileSpec *utils.File) (successCount, failedCount int, err error) {
	log.Info("Searching artifacts...")
	resultItems, err := utils.AqlSearchBySpec(fileSpec, mc, mc.client)
	if err != nil {
		return
	}
	successCount, failedCount, err = mc.moveFiles("", resultItems, fileSpec)
	return
}

func (mc *MoveCopyService) moveWildcard(fileSpec *utils.File) (successCount, failedCount int, err error) {
	log.Info("Searching artifacts...")
	fileSpec.IncludeDirs = "true"
	resultItems, err := utils.AqlSearchDefaultReturnFields(fileSpec, mc, mc.client)
	if err != nil {
		return
	}
	regexpPath := cliutils.PathToRegExp(fileSpec.Pattern)
	successCount, failedCount, err = mc.moveFiles(regexpPath, resultItems, fileSpec)
	return
}

func reduceMovePaths(resultItems []utils.AqlSearchResultItem, fileSpec *utils.File) []utils.AqlSearchResultItem {
	if strings.ToLower(fileSpec.Flat) == "true" {
		return utils.ReduceDirResult(resultItems, utils.FilterBottomChainResults)
	}
	return utils.ReduceDirResult(resultItems, utils.FilterTopChainResults)
}

func (mc *MoveCopyService) moveFiles(regexpPath string, resultItems []utils.AqlSearchResultItem, fileSpec *utils.File) (successCount, failedCount int, err error) {
	successCount = 0
	failedCount = 0
	resultItems = reduceMovePaths(resultItems, fileSpec)
	utils.LogSearchResults(len(resultItems))
	for _, v := range resultItems {
		destPathLocal := fileSpec.Target
		isFlat, e := cliutils.StringToBool(fileSpec.Flat, false)
		if e != nil {
			err = e
			return
		}
		if !isFlat {
			if strings.Contains(destPathLocal, "/") {
				file, dir := fileutils.GetFileAndDirFromPath(destPathLocal)
				destPathLocal = cliutils.TrimPath(dir + "/" + v.Path + "/" + file)
			} else {
				destPathLocal = cliutils.TrimPath(destPathLocal + "/" + v.Path + "/")
			}
		}
		destFile, e := cliutils.ReformatRegexp(regexpPath, v.GetFullUrl(), destPathLocal)
		if e != nil {
			err = e
			return
		}
		if strings.HasSuffix(destFile, "/") {
			if v.Type != "folder" {
				destFile += v.Name
			} else {
				mc.createPathForMoveAction(destFile)
			}
		}
		success, e := mc.moveFile(v.GetFullUrl(), destFile)
		if e != nil {
			err = e
			return
		}

		successCount += cliutils.Bool2Int(success)
		failedCount += cliutils.Bool2Int(!success)
	}
	return
}

func (mc *MoveCopyService) moveFile(sourcePath, destPath string) (bool, error) {
	message := moveMsgs[mc.moveType].MovingMsg + " artifact: " + sourcePath + " to: " + destPath
	if mc.IsDryRun() == true {
		log.Info("[Dry run] ", message)
		return true, nil
	}

	log.Info(message)

	moveUrl := mc.GetArtifactoryDetails().Url
	restApi := "api/" + string(mc.moveType) + "/" + sourcePath
	requestFullUrl, err := utils.BuildArtifactoryUrl(moveUrl, restApi, map[string]string{"to": destPath})
	if err != nil {
		return false, err
	}
	httpClientsDetails := mc.GetArtifactoryDetails().CreateArtifactoryHttpClientDetails()
	resp, body, err := httputils.SendPost(requestFullUrl, nil, httpClientsDetails)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 200 {
		log.Error("Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body))
	}

	log.Debug("Artifactory response:", resp.Status)
	return resp.StatusCode == 200, nil
}

// Create destPath in Artifactory
func (mc *MoveCopyService) createPathForMoveAction(destPath string) (bool, error) {
	if mc.IsDryRun() == true {
		log.Info("[Dry run] ", "Create path:", destPath)
		return true, nil
	}

	return createPathInArtifactory(destPath, mc)
}

func createPathInArtifactory(destPath string, conf utils.CommonConf) (bool, error) {
	rtUrl := conf.GetArtifactoryDetails().Url
	requestFullUrl, err := utils.BuildArtifactoryUrl(rtUrl, destPath, map[string]string{})
	if err != nil {
		return false, err
	}
	httpClientsDetails := conf.GetArtifactoryDetails().CreateArtifactoryHttpClientDetails()
	resp, body, err := httputils.SendPut(requestFullUrl, nil, httpClientsDetails)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 201 {
		log.Error("Artifactory response: " + resp.Status + "\n" + cliutils.IndentJson(body))
	}

	log.Debug("Artifactory response:", resp.Status)
	return resp.StatusCode == 200, nil
}

var moveMsgs = map[MoveType]MoveOptions{
	MOVE: MoveOptions{MovingMsg: "Moving", MovedMsg: "Moved"},
	COPY: MoveOptions{MovingMsg: "Copying", MovedMsg: "Copied"},
}

type MoveOptions struct {
	MovingMsg string
	MovedMsg  string
}

type MoveType string

type MoveCopyParams interface {
	utils.FileGetter
	GetFile() *utils.File
}

type MoveCopyParamsImpl struct {
	*utils.File
}

func (ds *MoveCopyParamsImpl) GetFile() *utils.File {
	return ds.File
}