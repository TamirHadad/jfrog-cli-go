package auth

import (
	"bytes"
	"encoding/json"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"golang.org/x/crypto/ssh"
	"errors"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
)

func sshAuthentication(url, sshKeyPath string) (string, map[string]string, error) {
	_, host, port, err := parseUrl(url)
	if err != nil {
	    return "", nil, err
	}

	log.Info("Performing SSH authentication...")
	if sshKeyPath == "" {
		err := cliutils.CheckError(errors.New("Cannot invoke the SshAuthentication function with no SSH key path. "))
        if err != nil {
			return "", nil, err
        }
	}

	buffer, err := ioutil.ReadFile(sshKeyPath)
	err = cliutils.CheckError(err)
	if err != nil {
		return "", nil, err
	}
	key, err := ssh.ParsePrivateKey(buffer)
	err = cliutils.CheckError(err)
	if err != nil {
		return "", nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User: "admin",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback : ssh.InsecureIgnoreHostKey(),
	}

	hostAndPort := host + ":" + strconv.Itoa(port)
	connection, err := ssh.Dial("tcp", hostAndPort, sshConfig)
	err = cliutils.CheckError(err)
	if err != nil {
		return "", nil, err
	}
	defer connection.Close()

	session, err := connection.NewSession()
	err = cliutils.CheckError(err)
	if err != nil {
		return "", nil, err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	err = cliutils.CheckError(err)
	if err != nil {
		return "", nil, err
	}

	var buf bytes.Buffer
	go io.Copy(&buf, stdout)

	session.Run("jfrog-authenticate")

	var result SshAuthResult
	err = json.Unmarshal(buf.Bytes(), &result)
	err = cliutils.CheckError(err)
	if err != nil {
		return "", nil, err
	}
	url = cliutils.AddTrailingSlashIfNeeded(result.Href)
	sshAuthHeaders := result.Headers
	log.Info("SSH authentication successful.")
	return url, sshAuthHeaders, nil
}

func parseUrl(url string) (protocol, host string, port int, err error) {
	pattern1 := "^(.+)://(.+):([0-9].+)/$"
	pattern2 := "^(.+)://(.+)$"

    var r *regexp.Regexp
	r, err = regexp.Compile(pattern1)
	err = cliutils.CheckError(err)
	if err != nil {
	    return
	}
	groups := r.FindStringSubmatch(url)
	if len(groups) == 4 {
		protocol = groups[1]
		host = groups[2]
		port, err = strconv.Atoi(groups[3])
		if err != nil {
			err = cliutils.CheckError(errors.New("URL: " + url + " is invalid. Expecting ssh://<host>:<port> or http(s)://..."))
		}
		return
	}

	r, err = regexp.Compile(pattern2)
	err = cliutils.CheckError(err)
	if err != nil {
	    return
	}
	groups = r.FindStringSubmatch(url)
	if len(groups) == 3 {
		protocol = groups[1]
		host = groups[2]
		port = 80
	}
	return
}

type SshAuthResult struct {
	Href    string
	Headers map[string]string
}
