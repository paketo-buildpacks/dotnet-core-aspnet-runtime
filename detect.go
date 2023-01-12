package dotnetcoreaspnetruntime

import (
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

type Environment struct {
	DotnetRollForward      string `env:"BP_DOTNET_ROLL_FORWARD"`
	DotnetFrameworkVersion string `env:"BP_DOTNET_FRAMEWORK_VERSION"`
}

func Detect(environment Environment, versionParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		var requirements []packit.BuildPlanRequirement

		if environment.DotnetFrameworkVersion != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
					"version":        environment.DotnetFrameworkVersion,
				},
			})
		}

		// check if the version is set in the buildpack.yml
		version, err := versionParser.ParseVersion(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        version,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "dotnet-core-aspnet-runtime"},
				},
				Requires: requirements,
			},
		}, nil
	}
}
