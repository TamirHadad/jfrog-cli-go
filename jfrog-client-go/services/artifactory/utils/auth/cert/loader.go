package cert

import (
	"net/http"
	"crypto/x509"
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"path/filepath"
	"crypto/tls"
	"io/ioutil"
)

func loadCertificates(caCertPool *x509.CertPool, certificatesDirPath string) error {
	if !fileutils.IsPathExists(certificatesDirPath) {
		return nil
	}
	files, err := ioutil.ReadDir(certificatesDirPath)
	err = cliutils.CheckError(err)
	if err != nil {
		return err
	}
	for _, file := range files {
		caCert, err := ioutil.ReadFile(filepath.Join(certificatesDirPath, file.Name()))
		err = cliutils.CheckError(err)
		if err != nil {
			return err
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}
	return nil
}

func GetTransportWithLoadedCert(certificatesDirPath string) (*http.Transport, error) {
	// Remove once SystemCertPool supports windows
	caCertPool, err := loadSystemRoots()
	err = cliutils.CheckError(err)
	if err != nil {
		return nil, err
	}
	err = loadCertificates(caCertPool, certificatesDirPath)
	if err != nil {
		return nil, err
	}
	// Setup HTTPS client
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
		ClientSessionCache: tls.NewLRUClientSessionCache(1)}
	tlsConfig.BuildNameToCertificate()
	return &http.Transport{TLSClientConfig: tlsConfig,Proxy: http.ProxyFromEnvironment}, nil

}