package commands

import (
	clientutils "github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
	"github.com/jfrogdev/jfrog-cli-go/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
)

type SearchResult struct {
	Path string `json:"path,omitempty"`
}

func Search(searchSpec *utils.SpecFiles, artDetails *config.ArtifactoryDetails) ([]SearchResult, error) {
	servicesManager, err := utils.CreateDefaultServiceManager(artDetails, false)
	if err != nil {
		return nil, err
	}
	log.Info("Searching artifacts...")
	var resultItems []clientutils.AqlSearchResultItem
	for i := 0; i < len(searchSpec.Files); i++ {
		currentSpec := searchSpec.Get(i)
		currentResultItems, err := servicesManager.Search(&clientutils.SearchParamsImpl{File: currentSpec})
		if err != nil {
			return nil, err
		}
		resultItems = append(resultItems, currentResultItems...)
	}

	result := aqlResultToSearchResult(resultItems)
	clientutils.LogSearchResults(len(resultItems))
	return result, err
}

func aqlResultToSearchResult(aqlResult []clientutils.AqlSearchResultItem) (result []SearchResult) {
	result = make([]SearchResult, len(aqlResult))
	for i, v := range aqlResult {
		tempResult := new(SearchResult)
		if v.Path != "." {
			tempResult.Path = v.Repo + "/" + v.Path + "/" + v.Name
		} else {
			tempResult.Path = v.Repo + "/" + v.Name
		}
		result[i] = *tempResult
	}
	return
}
