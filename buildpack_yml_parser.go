package dotnetcoreaspnetruntime

import (
	"os"

	"gopkg.in/yaml.v2"
)

type BuildpackYMLParser struct{}

func NewBuildpackYMLParser() BuildpackYMLParser {
	return BuildpackYMLParser{}
}

func (p BuildpackYMLParser) ParseVersion(path string) (string, error) {
	var buildpack struct {
		DotnetFramework struct {
			Version string `yaml:"version"`
		} `yaml:"dotnet-framework"`
	}

	file, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	defer file.Close()

	if !os.IsNotExist(err) {
		err = yaml.NewDecoder(file).Decode(&buildpack)
		if err != nil {
			return "", err
		}
	}

	return buildpack.DotnetFramework.Version, nil
}
