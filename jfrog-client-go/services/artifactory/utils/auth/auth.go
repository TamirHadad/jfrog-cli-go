package auth

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/io/httputils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
)

type ArtifactoryAuthConfiguration struct {
	Url            string            `json:"-"`
	User           string            `json:"-"`
	Password       string            `json:"-"`
	ApiKey         string            `json:"-"`
	SshKeysPath    string            `json:"-"`
	SshAuthHeaders map[string]string `json:"-"`
}

func (rt *ArtifactoryAuthConfiguration) SshAuthentication() (map[string]string, error) {
	if rt.SshKeysPath == "" {
		return  nil, nil
	}
	baseUrl, sshHeaders, err := sshAuthentication(rt.Url, rt.SshKeysPath)
	if err != nil {
		return nil, err
	}
	rt.Url = baseUrl
	return sshHeaders ,nil
}

func (rt *ArtifactoryAuthConfiguration) GetSshAuthHeaders() map[string]string {
	return rt.SshAuthHeaders
}

func (rt *ArtifactoryAuthConfiguration) GetUser() string {
	return rt.User
}

func (rt *ArtifactoryAuthConfiguration) CreateArtifactoryHttpClientDetails() httputils.HttpClientDetails {
	return httputils.HttpClientDetails{
		User:      rt.User,
		Password:  rt.Password,
		ApiKey:    rt.ApiKey,
		Headers:   cliutils.CopyMap(rt.SshAuthHeaders)}
}