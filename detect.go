package dotnetcoreaspnetruntime

import (
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type Environment struct {
	DotnetRollForward                string `env:"BP_DOTNET_ROLL_FORWARD"`
	DotnetRuntimeVersion             string `env:"BP_DOTNET_RUNTIME_VERSION"`
	DeprecatedDotnetFrameworkVersion string `env:"BP_DOTNET_FRAMEWORK_VERSION"`
}

func Detect(environment Environment, logger scribe.Emitter) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		var requirements []packit.BuildPlanRequirement

		if environment.DotnetRuntimeVersion != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_RUNTIME_VERSION",
					"version":        environment.DotnetRuntimeVersion,
				},
			})
		}

		if environment.DeprecatedDotnetFrameworkVersion != "" {
			logger.Subprocess(scribe.YellowColor("WARNING: BP_DOTNET_FRAMEWORK_VERSION is deprecated and will be removed in a future version. Please use BP_DOTNET_RUNTIME_VERSION instead."))
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
					"version":        environment.DeprecatedDotnetFrameworkVersion,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "dotnet-core-aspnet-runtime"},
				},
				Requires: requirements,
				Or: []packit.BuildPlan{
					{
						Provides: []packit.BuildPlanProvision{
							{Name: "dotnet-runtime"},
						},
					},
					{
						Provides: []packit.BuildPlanProvision{
							{Name: "dotnet-runtime"},
							{Name: "dotnet-aspnetcore"},
						},
					},
				},
			},
		}, nil
	}
}
