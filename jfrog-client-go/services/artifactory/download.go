package artifactory

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/types"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"errors"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"github.com/gofrog/parallel"
	"path"
	"path/filepath"
	"os"
	"sort"
	"github.com/jfrogdev/jfrog-cli-go/errors/httperrors"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
)

type DownloadService struct {
	client *helpers.JfrogHttpClient
	ArtDetails      *auth.ArtifactoryAuthConfiguration
	DryRun          bool
	Threads         int
	MinSplitSize    int64
	SplitCount      int
}

func (ds *DownloadService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return ds.ArtDetails
}

func (ds *DownloadService) SetArtifactoryDetails(rt *auth.ArtifactoryAuthConfiguration) {
	ds.ArtDetails = rt
}

func (ds *DownloadService) IsDryRun() bool {
	return ds.DryRun
}

func (ds *DownloadService) GetThreads() int {
	return ds.Threads
}

func (ds *DownloadService) SetThreads(threads int) {
	ds.Threads = threads
}

func (ds *DownloadService) SetArtDetails(artDetails *auth.ArtifactoryAuthConfiguration) {
	ds.ArtDetails = artDetails
}

func (ds *DownloadService) SetDryRun(isDryRun bool) {
	ds.DryRun = isDryRun
}

func (ds *DownloadService) setMinSplitSize(minSplitSize int64) {
	ds.MinSplitSize = minSplitSize
}

func NewDownloadService(client *helpers.JfrogHttpClient) *DownloadService {
	return &DownloadService{client:client}
}

func (ds *DownloadService) DownloadFiles(downloadParams DownloadParams) ([]utils.DependenciesBuildInfo, error) {
	buildDependencies := make([][]utils.DependenciesBuildInfo, ds.GetThreads())
	producerConsumer := parallel.NewBounedRunner(ds.GetThreads(), true)
	errorsQueue := utils.NewErrorsQueue(1)
	fileHandlerFunc := ds.createFileHandlerFunc(buildDependencies, downloadParams)
	log.Info("Searching items to download...")
	ds.prepareTasks(producerConsumer, fileHandlerFunc, errorsQueue, downloadParams)
	err := performTasks(producerConsumer, errorsQueue)
	if err != nil {
		return nil, err
	}
	return stripThreadIdFromBuildInfoDependencies(buildDependencies), err
}

func stripThreadIdFromBuildInfoDependencies(dependenciesBuildInfo [][]utils.DependenciesBuildInfo) []utils.DependenciesBuildInfo {
	var buildInfo []utils.DependenciesBuildInfo
	for _, v := range dependenciesBuildInfo {
		buildInfo = append(buildInfo, v...)
	}
	return buildInfo
}

func (ds *DownloadService) prepareTasks(producer parallel.Runner, fileContextHandler fileHandlerFunc, errorsQueue *utils.ErrorsQueue, downloadParams DownloadParams) {
	go func() {
		defer producer.Done()
		var err error
		var resultItems []utils.AqlSearchResultItem
		switch downloadParams.GetSpecType() {
		case utils.WILDCARD, utils.SIMPLE:
			resultItems, err = ds.collectFilesUsingWildcardPattern(downloadParams)
		case utils.AQL:
			resultItems, err = utils.AqlSearchBySpec(downloadParams.GetFile(), ds, ds.client)
		}

		if err != nil {
			errorsQueue.AddError(err)
			return
		}

		err = produceTasks(resultItems, downloadParams.GetFile(), producer, fileContextHandler, errorsQueue)
		if err != nil {
			errorsQueue.AddError(err)
			return
		}
	}()
}

func (ds *DownloadService) collectFilesUsingWildcardPattern(downloadParams DownloadParams) ([]utils.AqlSearchResultItem, error) {
	return utils.AqlSearchDefaultReturnFields(downloadParams.GetFile(), ds, ds.client)
}

func produceTasks(items []utils.AqlSearchResultItem, fileSpec *utils.File, producer parallel.Runner, fileHandler fileHandlerFunc, errorsQueue *utils.ErrorsQueue) error {
	flat, err := cliutils.StringToBool(fileSpec.Flat, false)
	if err != nil {
		return err
	}
	// Collect all folders path which might be needed to create.
	// key = folder path, value = the necessary data for producing create folder task.
	directoriesData := make(map[string]DownloadData)
	// Store all the paths which was created implicitly due to file upload.
	alreadyCreatedDirs := make(map[string]bool)
	// Store all the keys of directoriesData as an array.
	var directoriesDataKeys []string
	for _, v := range items {
		tempData := DownloadData{
			Dependency: v,
			DownloadPath: fileSpec.Pattern,
			Target: fileSpec.Target,
			Flat: flat,
		}
		if v.Type != "folder" {
			// Add a task, task is a function of type TaskFunc which later on will be executed by other go routine, the communication is done using channels.
			// The second argument is a error handling func in case the taskFunc return an error.
			producer.AddTaskWithError(fileHandler(tempData), errorsQueue.AddError)
			// We don't want to create directories which are created explicitly by download files when the --include-dirs flag is used.
			alreadyCreatedDirs[v.Path] = true
		} else {
			directoriesData, directoriesDataKeys = collectDirPathsToCreate(v, directoriesData, tempData, directoriesDataKeys)
		}
	}

	addCreateDirsTasks(directoriesDataKeys, alreadyCreatedDirs, producer, fileHandler, directoriesData, errorsQueue, flat)
	return nil
}

// Extract for the aqlResultItem the directory path, store the path the directoriesDataKeys and in the directoriesData map.
// In addition directoriesData holds the correlate DownloadData for each key, later on this DownloadData will be used to create a create dir tasks if needed.
// This function append the new data to directoriesDataKeys and to directoriesData and return the new map and the new []string
// We are storing all the keys of directoriesData in additional array(directoriesDataKeys) so we could sort the keys and access the maps in the sorted order.
func collectDirPathsToCreate(aqlResultItem utils.AqlSearchResultItem, directoriesData map[string]DownloadData, tempData DownloadData, directoriesDataKeys []string) (map[string]DownloadData, []string) {
	key := aqlResultItem.Name
	if aqlResultItem.Path != "." {
		key = aqlResultItem.Path + "/" + aqlResultItem.Name
	}
	directoriesData[key] = tempData
	directoriesDataKeys = append(directoriesDataKeys, key)
	return directoriesData, directoriesDataKeys
}

func addCreateDirsTasks(directoriesDataKeys []string, alreadyCreatedDirs map[string]bool, producer parallel.Runner, fileHandler fileHandlerFunc, directoriesData map[string]DownloadData, errorsQueue *utils.ErrorsQueue, isFlat bool) {
	// Longest path first
	// We are going to create the longest path first by doing so all sub paths of the longest path will be created implicitly.
	sort.Sort(sort.Reverse(sort.StringSlice(directoriesDataKeys)))
	for index, v := range directoriesDataKeys {
		// In order to avoid duplication we need to check the path wasn't already created by the previous action.
		if v != "." && // For some files the returned path can be the root path, ".", in that case we doing need to create any directory.
			(index == 0 || !utils.IsSubPath(directoriesDataKeys, index, "/")) {// directoriesDataKeys store all the path which might needed to be created, that's include duplicated paths.
			// By sorting the directoriesDataKeys we can assure that the longest path was created and therefore no need to create all it's sub paths.

			// Some directories were created due to file download when we aren't in flat download flow.
			if isFlat {
				producer.AddTaskWithError(fileHandler(directoriesData[v]), errorsQueue.AddError)
			} else if !alreadyCreatedDirs[v] {
				producer.AddTaskWithError(fileHandler(directoriesData[v]), errorsQueue.AddError)
			}
		}
	}
}

func performTasks(consumer parallel.Runner, errorsQueue *utils.ErrorsQueue) (err error) {
	// Blocked until finish consuming
	consumer.Run()
	err = errorsQueue.GetError()
	return
}

func createBuildDependencyItem(resultItem utils.AqlSearchResultItem) utils.DependenciesBuildInfo {
	return utils.DependenciesBuildInfo{
		Id: resultItem.Name,
		BuildInfoCommon : &utils.BuildInfoCommon{
			Sha1: resultItem.Actual_Sha1,
			Md5: resultItem.Actual_Md5,
		},
	}
}

func createDownloadFileDetails(downloadPath, localPath, localFileName string, acceptRanges *types.BoolEnum, size int64) (details *DownloadFileDetails) {
	details = &DownloadFileDetails{
		DownloadPath: downloadPath,
		LocalPath: localPath,
		LocalFileName: localFileName,
		AcceptRanges: acceptRanges,
		Size: size}
	return
}

func (ds *DownloadService) getFileRemoteDetails(downloadPath string, downloadParams DownloadParams) (*fileutils.FileDetails, error) {
	httpClientsDetails := ds.ArtDetails.CreateArtifactoryHttpClientDetails()
	details, err := ds.client.GetRemoteFileDetails(downloadPath, httpClientsDetails)
	if err != nil {
		err = cliutils.CheckError(errors.New("Artifactory " + err.Error()))
		if err != nil {
			return details, err
		}
	}
	return details, nil
}

func (ds *DownloadService) downloadFile(downloadFileDetails *DownloadFileDetails, logMsgPrefix string, downloadParams DownloadParams) error {
	httpClientsDetails := ds.ArtDetails.CreateArtifactoryHttpClientDetails()
	bulkDownload := ds.SplitCount == 0 || ds.MinSplitSize < 0 || ds.MinSplitSize * 1000 > downloadFileDetails.Size
	if !bulkDownload {
		acceptRange, err := ds.isFileAcceptRange(downloadFileDetails, downloadParams)
		if err != nil {
			return err
		}
		bulkDownload = !acceptRange
	}
	if bulkDownload {
		resp, err := ds.client.DownloadFile(downloadFileDetails.DownloadPath, downloadFileDetails.LocalPath, downloadFileDetails.LocalFileName, httpClientsDetails)
		// Ignore response status errors to continue downloading
		if err != nil && !httperrors.IsResponseStatusError(err) {
			return err
		}
		log.Debug(logMsgPrefix, "Artifactory response:", resp.Status)
	} else {
		concurrentDownloadFlags := helpers.ConcurrentDownloadFlags{
			DownloadPath: downloadFileDetails.DownloadPath,
			FileName:     downloadFileDetails.LocalFileName,
			LocalPath:    downloadFileDetails.LocalPath,
			FileSize:     downloadFileDetails.Size,
			SplitCount:   ds.SplitCount}

		ds.client.DownloadFileConcurrently(concurrentDownloadFlags, logMsgPrefix, httpClientsDetails)
	}
	return nil
}

func (ds *DownloadService) isFileAcceptRange(downloadFileDetails *DownloadFileDetails, downloadParams DownloadParams) (bool, error) {
	if downloadFileDetails.AcceptRanges == nil {
		details, err := ds.getFileRemoteDetails(downloadFileDetails.DownloadPath, downloadParams)
		if err != nil {
			return false, err
		}
		return details.AcceptRanges.GetValue(), nil
	}
	return downloadFileDetails.AcceptRanges.GetValue(), nil
}

func shouldDownloadFile(localFilePath, md5, sha1 string) (bool, error) {
	exists, err := fileutils.IsFileExists(localFilePath)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}
	localFileDetails, err := fileutils.GetFileDetails(localFilePath)
	if err != nil {
		return false, err
	}
	return localFileDetails.Checksum.Md5 != md5 || localFileDetails.Checksum.Sha1 != sha1, nil
}

func removeIfSymlink(localSymlinkPath string) error {
	if fileutils.IsPathSymlink(localSymlinkPath) {
		if err := os.Remove(localSymlinkPath); cliutils.CheckError(err) != nil {
			return err
		}
	}
	return nil
}

func createLocalSymlink(localPath, localFileName, symlinkArtifact string, symlinkChecksum bool, symlinkContentChecksum string, logMsgPrefix string) error {
	if symlinkChecksum && symlinkContentChecksum != "" {
		if !fileutils.IsPathExists(symlinkArtifact) {
			return cliutils.CheckError(errors.New("Symlink validation failed, target doesn't exist: " + symlinkArtifact))
		}
		sha1, err := fileutils.CalcSha1(symlinkArtifact)
		if err != nil {
			return err
		}
		if sha1 != symlinkContentChecksum {
			return cliutils.CheckError(errors.New("Symlink validation failed for target: " + symlinkArtifact))
		}
	}
	localSymlinkPath := filepath.Join(localPath, localFileName)
	isFileExists, err := fileutils.IsFileExists(localSymlinkPath)
	if err != nil {
		return err
	}
	// We can't create symlink in case a file with the same name already exist, we must remove the file before creating the symlink
	if isFileExists {
		if err := os.Remove(localSymlinkPath); err != nil {
			return err
		}
	}
	// Need to prepare the directories hierarchy
	_, err = fileutils.CreateFilePath(localPath, localFileName)
	if err != nil {
		return err
	}
	err = os.Symlink(symlinkArtifact, localSymlinkPath)
	if cliutils.CheckError(err) != nil {
		return err
	}
	log.Debug(logMsgPrefix, "Creating symlink file.")
	return nil
}

func getArtifactPropertyByKey(properties []utils.Property, key string) string {
	for _, v := range properties {
		if v.Key == key {
			return v.Value
		}
	}
	return ""
}

func getArtifactSymlinkPath(properties []utils.Property) string {
	return getArtifactPropertyByKey(properties, utils.ARTIFACTORY_SYMLINK)
}

func getArtifactSymlinkChecksum(properties []utils.Property) string {
	return getArtifactPropertyByKey(properties, utils.SYMLINK_SHA1)
}

type fileHandlerFunc func(DownloadData) parallel.TaskFunc

func (ds *DownloadService) createFileHandlerFunc(buildDependencies [][]utils.DependenciesBuildInfo, downloadParams DownloadParams) fileHandlerFunc {
	return func(downloadData DownloadData) parallel.TaskFunc {
		return func(threadId int) error {
			logMsgPrefix := cliutils.GetLogMsgPrefix(threadId, ds.DryRun)
			dependency := createBuildDependencyItem(downloadData.Dependency)
			downloadPath, e := utils.BuildArtifactoryUrl(ds.ArtDetails.Url, downloadData.Dependency.GetFullUrl(), make(map[string]string))
			if e != nil {
				return e
			}
			log.Info(logMsgPrefix + "Downloading", downloadData.Dependency.GetFullUrl())
			if ds.DryRun {
				return nil
			}

			regexpPattern := cliutils.PathToRegExp(downloadData.DownloadPath)
			placeHolderTarget, e := cliutils.ReformatRegexp(regexpPattern, downloadData.Dependency.GetFullUrl(), downloadData.Target)
			if e != nil {
				return e
			}
			localPath, localFileName := fileutils.GetLocalPathAndFile(downloadData.Dependency.Name, downloadData.Dependency.Path, placeHolderTarget, downloadData.Flat)
			if downloadData.Dependency.Type == "folder" {
				return createDir(localPath, localFileName, logMsgPrefix)
			}
			removeIfSymlink(filepath.Join(localPath, localFileName))
			if downloadParams.IsSymlink() {
				if isSymlink, e := createSymlinkIfNeeded(localPath, localFileName, logMsgPrefix, downloadData, buildDependencies, threadId, downloadParams); isSymlink {
					return e
				}
			}
			e = ds.downloadFileIfNeeded(downloadPath, localPath, localFileName, logMsgPrefix, downloadData, downloadParams)
			if e != nil {
				return e
			}
			buildDependencies[threadId] = append(buildDependencies[threadId], dependency)
			return nil
		}
	}
}

func (ds *DownloadService) downloadFileIfNeeded(downloadPath, localPath, localFileName, logMsgPrefix string, downloadData DownloadData, downloadParams DownloadParams) error {
	shouldDownload, e := shouldDownloadFile(path.Join(localPath, localFileName), downloadData.Dependency.Actual_Md5, downloadData.Dependency.Actual_Sha1)
	if e != nil {
		return e
	}
	if !shouldDownload {
		log.Debug(logMsgPrefix, "File already exists locally.")
		return nil
	}
	downloadFileDetails := createDownloadFileDetails(downloadPath, localPath, localFileName, nil, downloadData.Dependency.Size)
	return ds.downloadFile(downloadFileDetails, logMsgPrefix, downloadParams)
}

func createDir(localPath, localFileName, logMsgPrefix string) error {
	folderPath := filepath.Join(localPath, localFileName)
	e := fileutils.CreateDirIfNotExist(folderPath)
	if e != nil {
		return e
	}
	log.Info(logMsgPrefix + "Creating folder: " + folderPath)
	return nil
}

func createSymlinkIfNeeded(localPath, localFileName, logMsgPrefix string, downloadData DownloadData, buildDependencies [][]utils.DependenciesBuildInfo, threadId int, downloadParams DownloadParams) (bool, error) {
	symlinkArtifact := getArtifactSymlinkPath(downloadData.Dependency.Properties)
	isSymlink := len(symlinkArtifact) > 0
	if isSymlink {
		symlinkChecksum := getArtifactSymlinkChecksum(downloadData.Dependency.Properties)
		if e := createLocalSymlink(localPath, localFileName, symlinkArtifact, downloadParams.ValidateSymlinks(), symlinkChecksum, logMsgPrefix); e != nil {
			return isSymlink, e
		}
		dependency := createBuildDependencyItem(downloadData.Dependency)
		buildDependencies[threadId] = append(buildDependencies[threadId], dependency)
		return isSymlink, nil
	}
	return isSymlink, nil
}

type DownloadFileDetails struct {
	DownloadPath  string          `json:"DownloadPath,omitempty"`
	LocalPath     string          `json:"LocalPath,omitempty"`
	LocalFileName string          `json:"LocalFileName,omitempty"`
	AcceptRanges  *types.BoolEnum `json:"AcceptRanges,omitempty"`
	Size          int64           `json:"Size,omitempty"`
}

type DownloadData struct {
	Dependency   utils.AqlSearchResultItem
	DownloadPath string
	Target       string
	Flat         bool
}

type DownloadParams interface {
	utils.FileGetter
	IsSymlink() bool
	ValidateSymlinks() bool
	GetFile() *utils.File
}

type DownloadParamsImpl struct {
	*utils.File
	Symlink         bool
	ValidateSymlink bool
}

func (ds *DownloadParamsImpl) GetFile() *utils.File {
	return ds.File
}

func (ds *DownloadParamsImpl) IsSymlink() bool {
	return ds.Symlink
}

func (ds *DownloadParamsImpl) ValidateSymlinks() bool {
	return ds.ValidateSymlink
}

