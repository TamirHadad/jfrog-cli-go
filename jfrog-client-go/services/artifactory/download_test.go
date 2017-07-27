package artifactory

import (
	"testing"
	"os"
	"path/filepath"
	"strings"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/tests"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	"io/ioutil"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
)

func uploadDummyFile(t *testing.T) {
	workingDir, _, err := tests.CreateFileWithContent("a.in", "/out/")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(workingDir)
	pattern := filepath.Join(workingDir, "*")
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	uploadService := NewUploadService(helpers.NewDefaultJforgHttpClient())
	uploadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	uploadService.Threads = 3
	up := &UploadParamsImp{}
	up.File = &utils.File{Pattern:pattern, Flat:"true", Target:targetRepo + "test/"}
	_, uploaded, failed, err := uploadService.UploadFiles(up)
	if uploaded != 1 {
		t.Error("Expected to upload 1 file.")
	}
	if failed != 0 {
		t.Error("Failed to upload", failed, "files.")
	}
	if err != nil {
		t.Error(err)
	}
	up.File = &utils.File{Pattern:pattern, Flat:"true", Target:targetRepo + "b.in"}
	_, uploaded, failed, err = uploadService.UploadFiles(up)
	if uploaded != 1 {
		t.Error("Expected to upload 1 file.")
	}
	if failed != 0 {
		t.Error("Failed to upload", failed, "files.")
	}
	if err != nil {
		t.Error(err)
	}
}

func TestFlatDownload(t *testing.T) {
	uploadDummyFile(t)
	var err error
	workingDir, err := ioutil.TempDir("", "downloadTests")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(workingDir)
	downloadPattern := targetRepo + "*"
	downloadTarget := workingDir + string(filepath.Separator)
	downloadService := NewDownloadService(helpers.NewDefaultJforgHttpClient())
	downloadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	downloadService.SetThreads(3)
	_, err = downloadService.DownloadFiles(&DownloadParamsImpl{File:&utils.File{Pattern: downloadPattern, Flat: "true", Target: downloadTarget}})
	if err != nil {
		t.Error(err)
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir, "a.in")) {
		t.Error("Missing file a.in")
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir, "b.in")) {
		t.Error("Missing file b.in")
	}

	workingDir2, err := ioutil.TempDir("", "downloadTests")
	downloadTarget = workingDir2 + string(filepath.Separator)
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(workingDir2)
	_, err = downloadService.DownloadFiles(&DownloadParamsImpl{File:&utils.File{Pattern: downloadPattern, Flat: "false", Target: downloadTarget}})
	if err != nil {
		t.Error(err)
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir2, "test", "a.in")) {
		t.Error("Missing file a.in")
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir2, "b.in")) {
		t.Error("Missing file b.in")
	}

	artifactoryCleanUp(err, getTestsCommonConf(), t)
}

func TestRecursiveDownload(t *testing.T) {
	uploadDummyFile(t)
	var err error
	workingDir, err := ioutil.TempDir("", "downloadTests")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(workingDir)
	downloadConfiguration := getTestsCommonConf()
	downloadPattern := targetRepo + "*"
	downloadTarget := workingDir + string(filepath.Separator)
	downloadService := NewDownloadService(helpers.NewDefaultJforgHttpClient())
	downloadService.ArtDetails = downloadConfiguration.GetArtifactoryDetails()
	downloadService.SetThreads(3)
	_, err = downloadService.DownloadFiles(&DownloadParamsImpl{File:&utils.File{Pattern: downloadPattern, Recursive: "true", Flat: "true", Target: downloadTarget}})
	if err != nil {
		t.Error(err)
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir, "a.in")) {
		t.Error("Missing file a.in")
	}

	if !fileutils.IsPathExists(filepath.Join(workingDir, "b.in")) {
		t.Error("Missing file b.in")
	}

	workingDir2, err := ioutil.TempDir("", "downloadTests")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(workingDir2)
	downloadTarget = workingDir2 + string(filepath.Separator)
	_, err = downloadService.DownloadFiles(&DownloadParamsImpl{File:&utils.File{Pattern: downloadPattern, Recursive: "false", Flat: "true", Target: downloadTarget}})
	if err != nil {
		t.Error(err)
	}
	if fileutils.IsPathExists(filepath.Join(workingDir2, "a.in")) {
		t.Error("Should not download a.in")
	}

	if !fileutils.IsPathExists(filepath.Join(workingDir2, "b.in")) {
		t.Error("Missing file b.in")
	}

	artifactoryCleanUp(err, downloadConfiguration, t)
}

func TestPlaceholderDownload(t *testing.T) {
	uploadDummyFile(t)
	var err error
	workingDir, err := ioutil.TempDir("", "downloadTests")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(workingDir)
	downloadConfiguration := getTestsCommonConf()
	downloadPattern := targetRepo + "(*).in"
	downloadTarget := workingDir + string(filepath.Separator) + "{1}" + string(filepath.Separator)
	downloadService := NewDownloadService(helpers.NewDefaultJforgHttpClient())
	downloadService.ArtDetails = downloadConfiguration.GetArtifactoryDetails()
	downloadService.SetThreads(3)
	_, err = downloadService.DownloadFiles(&DownloadParamsImpl{File:&utils.File{Pattern: downloadPattern, Recursive: "true", Flat: "true", Target: downloadTarget}})
	if err != nil {
		t.Error(err)
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir, "test", "a", "a.in")) {
		t.Error("Missing file a.in")
	}

	if !fileutils.IsPathExists(filepath.Join(workingDir, "b", "b.in")) {
		t.Error("Missing file b.in")
	}

	artifactoryCleanUp(err, downloadConfiguration, t)
}

func TestIncludeDirsDownload(t *testing.T) {
	uploadDummyFile(t)
	var err error
	workingDir, err := ioutil.TempDir("", "downloadTests")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(workingDir)
	downloadConfiguration := getTestsCommonConf()
	downloadPattern := targetRepo + "*"
	downloadTarget := workingDir + string(filepath.Separator)
	downloadService := NewDownloadService(helpers.NewDefaultJforgHttpClient())
	downloadService.ArtDetails = downloadConfiguration.GetArtifactoryDetails()
	downloadService.SetThreads(3)
	_, err = downloadService.DownloadFiles(&DownloadParamsImpl{File:&utils.File{Pattern: downloadPattern, IncludeDirs:"true", Recursive: "false", Flat: "false", Target: downloadTarget}})
	if err != nil {
		t.Error(err)
	}
	if !fileutils.IsPathExists(filepath.Join(workingDir, "test")) {
		t.Error("Missing test folder")
	}

	if !fileutils.IsPathExists(filepath.Join(workingDir, "b.in")) {
		t.Error("Missing file b.in")
	}

	artifactoryCleanUp(err, downloadConfiguration, t)
}

