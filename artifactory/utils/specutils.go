package utils

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/io/fileutils"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"encoding/json"
	"strconv"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client-go/services/artifactory/utils"
	"fmt"
	"bytes"
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils/log"
)

type SpecFiles struct {
	Files []utils.File
}

func (spec *SpecFiles) Get(index int) *utils.File {
	if index < len(spec.Files) {
		return &spec.Files[index]
	}
	return new(utils.File)
}

func CreateSpecFromFile(specFilePath string, specVars map[string]string) (spec *SpecFiles, err error) {
	spec = new(SpecFiles)
	content, err := fileutils.ReadFile(specFilePath)
	if cliutils.CheckError(err) != nil {
		return
	}

	if len(specVars) > 0 {
		content = replaceSpecVars(content, specVars)
	}

	err = json.Unmarshal(content, spec)
	if cliutils.CheckError(err) != nil {
		return
	}
	return
}

func replaceSpecVars(content []byte, specVars map[string]string) []byte {
	log.Debug("Replacing variables in the provided File Spec: \n" + string(content))
	for key, val := range specVars {
		key = "${" + key + "}"
		log.Debug(fmt.Sprintf("Replacing '%s' with '%s'", key, val))
		content = bytes.Replace(content, []byte(key), []byte(val), -1)
	}
	log.Debug("The reformatted File Spec is: \n" + string(content))
	return content
}

func CreateSpec(pattern, target, props, build string, recursive, flat, regexp, includeDirs bool) (spec *SpecFiles) {
	spec = &SpecFiles{
		Files: []utils.File{
			{
				Pattern:     pattern,
				Target:      target,
				Props:       props,
				Build:       build,
				Recursive:   strconv.FormatBool(recursive),
				Flat:        strconv.FormatBool(flat),
				Regexp:      strconv.FormatBool(regexp),
				IncludeDirs: strconv.FormatBool(includeDirs),
			},
		},
	}
	return spec
}