package utils

type BuildInfoCommon struct {
	Sha1 string `json:"sha1,omitempty"`
	Md5  string `json:"md5,omitempty"`
}

type ArtifactsBuildInfo struct {
	Name string `json:"name,omitempty"`
	*BuildInfoCommon
}

type DependenciesBuildInfo struct {
	Id string `json:"id,omitempty"`
	*BuildInfoCommon
}