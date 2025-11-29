package dotnetcoreaspnetruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gravityblast/go-jsmin"
)

type framework struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RuntimeConfigParser struct{}

func NewRuntimeConfigParser() RuntimeConfigParser {
	return RuntimeConfigParser{}
}

func (p RuntimeConfigParser) Parse(glob string) (string, error) {
	files, err := filepath.Glob(glob)
	if err != nil {
		return "", fmt.Errorf("%w: %q", err, glob)
	}

	if len(files) > 1 {
		return "", fmt.Errorf("multiple *.runtimeconfig.json files present: %v", files)
	}

	if len(files) == 0 {
		return "", os.ErrNotExist
	}

	path := files[0]

	var data struct {
		RuntimeOptions struct {
			Framework  framework   `json:"framework"`
			Frameworks []framework `json:"frameworks"`
		} `json:"runtimeOptions"`
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	buffer := bytes.NewBuffer(nil)
	err = jsmin.Min(file, buffer)
	if err != nil {
		return "", err
	}

	err = json.NewDecoder(buffer).Decode(&data)
	if err != nil {
		return "", err
	}

	var frameworkVersion string

	switch data.RuntimeOptions.Framework.Name {
	case "Microsoft.NETCore.App":
		frameworkVersion = versionOrWildcard(data.RuntimeOptions.Framework.Version)
	case "Microsoft.AspNetCore.App":
		frameworkVersion = versionOrWildcard(data.RuntimeOptions.Framework.Version)
	}

	var runtimeV string
	var aspnetV string
	for _, f := range data.RuntimeOptions.Frameworks {
		switch f.Name {
		case "Microsoft.NETCore.App":
			if runtimeV != "" || frameworkVersion != "" {
				return "", fmt.Errorf("malformed runtimeconfig.json: multiple '%s' frameworks specified", f.Name)
			}
			runtimeV = versionOrWildcard(f.Version)
		case "Microsoft.AspNetCore.App":
			if aspnetV != "" || frameworkVersion != "" {
				return "", fmt.Errorf("malformed runtimeconfig.json: multiple '%s' frameworks specified", f.Name)
			}
			aspnetV = versionOrWildcard(f.Version)
		default:
			continue
		}
	}

	if frameworkVersion == "" {
		if runtimeV != "" {
			frameworkVersion = runtimeV
		}
		// ASP.NET Runtime version gets higher priority
		if aspnetV != "" {
			frameworkVersion = aspnetV
		}
	}

	return frameworkVersion, nil
}

func versionOrWildcard(version string) string {
	if version == "" {
		return "*"
	}
	return version
}
