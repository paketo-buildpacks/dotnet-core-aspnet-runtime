package dotnetcoreaspnetruntime

import "github.com/paketo-buildpacks/packit/v2"

type Environment struct {
	DotnetRollForward      string `env:"BP_DOTNET_ROLL_FORWARD"`
	DotnetFrameworkVersion string `env:"BP_DOTNET_FRAMEWORK_VERSION"`
}

func Detect(environment Environment) packit.DetectFunc {
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
