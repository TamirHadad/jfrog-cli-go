package utils

import (
	"github.com/jfrogdev/jfrog-cli-go/utils/cliutils"
	"strings"
)

const (
	WILDCARD SpecType = "wildcard"
	SIMPLE SpecType = "simple"
	AQL SpecType = "aql"
)

type Aql struct {
	ItemsFind string `json:"items.find"`
}

type File struct {
	Pattern     string
	Target      string
	Props       string
	Recursive   string
	Flat        string
	Regexp      string
	Aql         Aql
	Build       string
	IncludeDirs string
}

type FileGetter interface {
	GetPattern() string
	SetPattern(pattern string)
	SetTarget(target string)
	GetTarget() string
	GetProps() string
	GetRecursive() string
	GetFlat() string
	GetRegexp() string
	GetAql() Aql
	GetBuild() string
	IsIncludeDirs() bool
	GetSpecType() (specType SpecType)
}

func (f *File) GetPattern() string {
	return f.Pattern
}

func (f *File) SetPattern(pattern string) {
	f.Pattern = pattern
}

func (f *File) SetTarget(target string) {
	f.Target = target
}

func (f *File) GetTarget() string {
	return f.Target
}

func (f *File) GetProps() string {
	return f.Props
}

func (f *File) GetRecursive() string {
	return f.Recursive
}

func (f *File) GetFlat() string {
	return f.Flat
}

func (f *File) GetRegexp() string {
	return f.Regexp
}

func (f *File) GetAql() Aql {
	return f.Aql
}

func (f *File) GetBuild() string {
	return f.Build
}

func (f File) IsIncludeDirs() bool {
	return f.IncludeDirs == "true"
}

func (f *File) SetProps(props string) {
	f.Props = props
}

func (aql *Aql) UnmarshalJSON(value []byte) error {
	str := string(value)
	first := strings.Index(str[strings.Index(str, "{") + 1 :], "{")
	last := strings.LastIndex(str, "}")

	aql.ItemsFind = cliutils.StripChars(str[first:last], "\n\t ")
	return nil
}

func (f File) GetSpecType() (specType SpecType) {
	switch {
	case f.Pattern != "" && (IsWildcardPattern(f.Pattern) || f.Build != ""):
		specType = WILDCARD
	case f.Pattern != "":
		specType = SIMPLE
	case f.Aql.ItemsFind != "" :
		specType = AQL
	}
	return specType
}

type SpecType string