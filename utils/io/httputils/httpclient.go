package httputils

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"net/http"
)

type HttpClientDetails struct {
	User          string
	Password      string
	ApiKey        string
	Headers       map[string]string
	Transport     *http.Transport
}

func (httpClientDetails HttpClientDetails) Clone() *HttpClientDetails {
	headers := make(map[string]string)
	cliutils.MergeMaps(httpClientDetails.Headers, headers)
	return &HttpClientDetails{
		User:      httpClientDetails.User,
		Password:  httpClientDetails.Password,
		ApiKey:    httpClientDetails.ApiKey,
		Headers:   headers}
}