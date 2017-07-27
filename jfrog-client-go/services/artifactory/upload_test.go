package artifactory

import (
	"testing"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/tests"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"path/filepath"
	"os"
	"strings"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	//"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"fmt"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
)

const rtUrl = "http://localhost:8081/artifactory/"
const targetRepo = "Copy/"
const rtUser = "admin"
const rtPassword = "password"

func TestFlatUpload(t *testing.T) {
	workingDir, _, err := tests.CreateFileWithContent("a.in", "/out/")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(workingDir)
	pattern := filepath.Join(workingDir, "out", "*")
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	uploadService := NewUploadService(helpers.NewDefaultJforgHttpClient())
	uploadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	uploadService.Threads = 3
	up := &UploadParamsImp{}
	up.File = &utils.File{Pattern:pattern, Flat:"true", Target:targetRepo}
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
	searchService := NewSearchService(helpers.NewDefaultJforgHttpClient())
	items, err := searchService.Search(&utils.File{Pattern: targetRepo}, getTestsCommonConf())
	if err != nil {
		t.Error(err)
	}
	if len(items) > 1 {
		t.Error("Expected single file.")
	}
	for _, item := range items {
		if item.Path != "." {
			t.Error("Expected path to be root due to using the flat flag.", "Got:", item.Path)
		}
	}
	artifactoryCleanUp(err, getTestsCommonConf(), t)
}

func TestRecursiveUpload(t *testing.T) {
	workingDir, _, err := tests.CreateFileWithContent("a.in", "/out/")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(workingDir)
	uploadConfiguration := getTestsCommonConf()
	pattern := filepath.Join(workingDir, "*")
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	uploadService := NewUploadService(helpers.NewDefaultJforgHttpClient())
	uploadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	uploadService.Threads = 3
	up := &UploadParamsImp{}
	up.File = &utils.File{Pattern:pattern, Recursive:"true", Target:targetRepo}
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
	searchService := NewSearchService(helpers.NewDefaultJforgHttpClient())
	items, err := searchService.Search(&utils.File{Pattern: targetRepo}, uploadConfiguration)
	if err != nil {
		t.Error(err)
	}
	if len(items) > 1 {
		t.Error("Expected single file.")
	}
	for _, item := range items {
		if item.Path != "." {
			t.Error("Expected path to be root(flat by default).", "Got:", item.Path)
		}
		if item.Name != "a.in" {
			t.Error("Missing File a.in")
		}
	}
	artifactoryCleanUp(err, uploadConfiguration, t)
}

func TestPlaceholderUpload(t *testing.T) {
	workingDir, _, err := tests.CreateFileWithContent("a.in", "/out/")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(workingDir)
	uploadConfiguration := getTestsCommonConf()
	pattern := filepath.Join(workingDir, "(*).in")
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	uploadService := NewUploadService(helpers.NewDefaultJforgHttpClient())
	uploadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	uploadService.Threads = 3
	up := &UploadParamsImp{}
	up.File = &utils.File{Pattern:pattern, Recursive:"true", Target:targetRepo + "{1}"}
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
	searchService := NewSearchService(helpers.NewDefaultJforgHttpClient())
	items, err := searchService.Search(&utils.File{Pattern: targetRepo}, uploadConfiguration)
	if err != nil {
		t.Error(err)
	}
	if len(items) > 1 {
		t.Error("Expected single file.")
	}
	for _, item := range items {
		if item.Path != "out" {
			t.Error("Expected path to be out.", "Got:", item.Path)
		}
		if item.Name != "a" {
			t.Error("Missing File a")
		}
	}
	artifactoryCleanUp(err, uploadConfiguration, t)
}

func TestIncludeDirsUpload(t *testing.T) {
	workingDir, _, err := tests.CreateFileWithContent("a.in", "/out/")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(workingDir)
	uploadConfiguration := getTestsCommonConf()
	pattern := filepath.Join(workingDir, "*")
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	uploadService := NewUploadService(helpers.NewDefaultJforgHttpClient())
	uploadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	uploadService.Threads = 3
	up := &UploadParamsImp{}
	up.File = &utils.File{Pattern:pattern, IncludeDirs:"true", Recursive:"false", Target:targetRepo}
	_, uploaded, failed, err := uploadService.UploadFiles(up)
	if uploaded != 0 {
		t.Error("Expected to upload 1 file.")
	}
	if failed != 0 {
		t.Error("Failed to upload", failed, "files.")
	}
	if err != nil {
		t.Error(err)
	}
	searchService := NewSearchService(helpers.NewDefaultJforgHttpClient())
	items, err := searchService.Search(&utils.File{Pattern: targetRepo, IncludeDirs: "true"}, uploadConfiguration)
	if err != nil {
		t.Error(err)
	}
	if len(items) < 2 {
		t.Error("Expected to get at least two items, default and the out folder.")
	}
	for _, item := range items {
		fmt.Println(item.Name)
		if item.Name == "." {
			continue
		}
		if item.Path != "." {
			t.Error("Expected path to be root(flat by default).", "Got:", item.Path)
		}
		if item.Name != "out" {
			t.Error("Missing directory out.")
		}
	}
	artifactoryCleanUp(err, uploadConfiguration, t)
}

func TestExplodeUpload(t *testing.T) {
	workingDir, filePath, err := tests.CreateFileWithContent("a.in", "/out/")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(workingDir)
	err = fileutils.ZipFolderFiles(filePath, filepath.Join(workingDir, "zipFile.zip"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = os.Remove(filePath)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	uploadConfiguration := getTestsCommonConf()
	pattern := filepath.Join(workingDir, "*.zip")
	pattern = strings.Replace(pattern, "\\", "\\\\", -1)
	uploadService := NewUploadService(helpers.NewDefaultJforgHttpClient())
	uploadService.ArtDetails = getTestsCommonConf().GetArtifactoryDetails()
	uploadService.Threads = 3
	up := &UploadParamsImp{}
	up.File = &utils.File{Pattern:pattern, IncludeDirs:"true", Recursive:"false", Target:targetRepo}
	up.ExplodeArchive = true
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
	searchService := NewSearchService(helpers.NewDefaultJforgHttpClient())
	items, err := searchService.Search(&utils.File{Pattern: targetRepo, IncludeDirs: "true"}, uploadConfiguration)
	if err != nil {
		t.Error(err)
	}
	if len(items) < 2 {
		t.Error("Expected to get at least two items, default and the out folder.")
	}
	for _, item := range items {
		if item.Name == "." {
			continue
		}
		if item.Name != "a.in" {
			t.Error("Missing file a.in")
		}
	}
	artifactoryCleanUp(err, uploadConfiguration, t)
}

func getTestsCommonConf() utils.CommonConf {
	auth := &auth.ArtifactoryAuthConfiguration{Url: rtUrl, User:rtUser, Password:rtPassword}
	common := &utils.CommonConfImpl{}
	common.SetArtifactoryDetails(auth)
	return common
}

func artifactoryCleanUp(err error, configuration utils.CommonConf, t *testing.T) {
	deleteService := NewDeleteService(helpers.NewDefaultJforgHttpClient())
	toDelete, err := deleteService.GetPathsToDelete(&utils.File{Pattern: targetRepo}, configuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = deleteService.DeleteFiles(toDelete, configuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
