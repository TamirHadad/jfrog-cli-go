package utils

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/config"
	"net/http"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils/auth/cert"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/helpers"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services"
)

func GetJfrogSecurityDir() (string, error) {
	confPath, err := config.GetJfrogHomeDir()
	if err != nil {
		return "", err
	}
	return confPath + "security/", nil
}

func GetEncryptedPasswordFromArtifactory(artifactoryAuth *auth.ArtifactoryAuthConfiguration) (*http.Response, string, error) {
	apiUrl := artifactoryAuth.Url + "api/security/encryptedPassword"
	httpClientsDetails := artifactoryAuth.CreateArtifactoryHttpClientDetails()
	securityDir, err := GetJfrogSecurityDir()
	if err != nil {
		return nil, "", err
	}
	transport, err := cert.GetTransportWithLoadedCert(securityDir)
	client := helpers.NewJforgHttpClient(&http.Client{Transport: transport})
	resp, body, _, err := client.SendGet(apiUrl, true, httpClientsDetails)
	return resp, string(body), err
}

func CreateDefaultServiceManager(artDetails *config.ArtifactoryDetails, isDryRun bool) (*services.ArtifactoryServicesManager, error) {
	certPath, err := GetJfrogSecurityDir()
	if err != nil {
		return nil, err
	}
	authConfig := artDetails.CreateArtAuthConfig()
	serviceConfig, err := (&services.ArtifactoryServicesConfigBuilder{}).
		SetArtDetails(authConfig).
		SetCertifactesPath(certPath).
		SetDryRun(isDryRun).
		Build()
	if err != nil {
		return nil, err
	}
	return services.NewArtifactoryService(serviceConfig)
}

//func CreateArtifactoryServicesManager(artDetails *config.ArtifactoryDetails) (*services.ArtifactoryServicesManager, error) {
//	securityDir, err := GetJfrogSecurityDir()
//	if err != nil {
//		return nil, err
//	}
//
//	return services.NewArtifactoryService(&services.artifactoryServicesConfig{artConfiguration: artDetails.CreateArtAuthConfig(), certifactesPath: securityDir})
//}
