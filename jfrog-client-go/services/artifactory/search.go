package artifactory

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
)

type SearchService struct {
	client *helpers.JfrogHttpClient
	ArtDetails *auth.ArtifactoryAuthConfiguration
}

func (s *SearchService) GetArtifactoryDetails() *auth.ArtifactoryAuthConfiguration {
	return s.ArtDetails
}

func (s *SearchService) SetArtifactoryDetails(rt *auth.ArtifactoryAuthConfiguration) {
	s.ArtDetails = rt
}

func (s *SearchService) IsDryRun() bool {
	return false
}

func NewSearchService(client *helpers.JfrogHttpClient) *SearchService {
	return &SearchService{client:client}
}

func (s *SearchService) Search(searchParamsImpl utils.SearchParams, conf utils.CommonConf) ([]utils.AqlSearchResultItem, error) {
	return utils.SearchBySpecFiles(searchParamsImpl, conf, s.client)
}