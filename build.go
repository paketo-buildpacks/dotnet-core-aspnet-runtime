package dotnetcoreaspnetruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface VersionResolver --output fakes/version_resolver.go
type VersionResolver interface {
	Resolve(path string, entry packit.BuildpackPlanEntry, stack string) (postal.Dependency, error)
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	GenerateFromDependency(dependency postal.Dependency, dir string) (sbom.SBOM, error)
}

//go:generate faux --interface ConfigParser --output fakes/config_parser.go
type ConfigParser interface {
	Parse(runtimeConfigFileGlob string) (string, error)
}

func Build(
	entries EntryResolver,
	dependencies DependencyManager,
	versionResolver VersionResolver,
	sbomGenerator SBOMGenerator,
	configParser ConfigParser,
	logger scribe.Emitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		// This no-op path is to accommodate for it the dotnet-publish buildpack
		// generates a self-contained app
		frameworkVersion, err := configParser.Parse(filepath.Join(context.WorkingDir, "*.runtimeconfig.json"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logger.Process("Skipping build process. No runtimeconfig.json found.")
				logger.Break()
				return packit.BuildResult{}, nil
			}
			return packit.BuildResult{}, err
		}

		if frameworkVersion == "" {
			logger.Process("Skipping build process. No required runtime frameworks listed in runtimeconfig.json. App is self-contained.")
			logger.Break()
			return packit.BuildResult{}, nil
		} else {
			context.Plan.Entries = append([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-aspnet-runtime",
					Metadata: map[string]interface{}{
						"version":        frameworkVersion,
						"version-source": "runtimeconfig.json",
					},
				},
			}, context.Plan.Entries...)
		}

		logger.Process("Resolving ASP.NET Core Runtime version")

		priorities := []interface{}{
			"BP_DOTNET_FRAMEWORK_VERSION",
			"buildpack.yml",
			"runtimeconfig.json",
			regexp.MustCompile(`.*\.(cs)|(fs)|(vb)proj`),
		}
		entry, sortedEntries := entries.Resolve("dotnet-core-aspnet-runtime", context.Plan.Entries, priorities)
		logger.Candidates(sortedEntries)

		source, _ := entry.Metadata["version-source"].(string)
		if source == "buildpack.yml" {
			nextMajorVersion := semver.MustParse(context.BuildpackInfo.Version).IncMajor()
			logger.Subprocess("WARNING: Setting the .NET Framework version through buildpack.yml will be deprecated soon in .NET Core ASP.NET Core Runtime Buildpack v%s.", nextMajorVersion.String())
			logger.Subprocess("Please specify the version through the $BP_DOTNET_FRAMEWORK_VERSION environment variable instead. See docs for more information.")
			logger.Break()
		}

		dependency, err := versionResolver.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())

		dotnetCoreAspnetRuntimeLayer, err := context.Layers.Get("dotnet-core-aspnet-runtime")
		if err != nil {
			return packit.BuildResult{}, err
		}

		bom := dependencies.GenerateBillOfMaterials(dependency)
		launch, build := entries.MergeLayerTypes("dotnet-core-aspnet-runtime", context.Plan.Entries)

		var buildMetadata packit.BuildMetadata
		if build {
			buildMetadata.BOM = bom
		}

		var launchMetadata packit.LaunchMetadata
		if launch {
			launchMetadata.BOM = bom
		}

		cachedChecksum, ok := dotnetCoreAspnetRuntimeLayer.Metadata["dependency-checksum"].(string)
		if ok && cargo.Checksum(dependency.Checksum).MatchString(cachedChecksum) {
			logger.Process(fmt.Sprintf("Reusing cached layer %s", dotnetCoreAspnetRuntimeLayer.Path))
			logger.Break()

			dotnetCoreAspnetRuntimeLayer.Launch, dotnetCoreAspnetRuntimeLayer.Build, dotnetCoreAspnetRuntimeLayer.Cache = launch, build, build

			return packit.BuildResult{
				Layers: []packit.Layer{dotnetCoreAspnetRuntimeLayer},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil

		}

		logger.Process("Executing build process")

		dotnetCoreAspnetRuntimeLayer, err = dotnetCoreAspnetRuntimeLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		dotnetCoreAspnetRuntimeLayer.Launch, dotnetCoreAspnetRuntimeLayer.Build, dotnetCoreAspnetRuntimeLayer.Cache = launch, build, build

		logger.Subprocess("Installing ASP.NET Core Runtime %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencies.Deliver(dependency, context.CNBPath, dotnetCoreAspnetRuntimeLayer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		dotnetCoreAspnetRuntimeLayer.Metadata = map[string]interface{}{
			"dependency-checksum": dependency.Checksum,
		}

		dotnetCoreAspnetRuntimeLayer.LaunchEnv.Prepend(
			"PATH",
			dotnetCoreAspnetRuntimeLayer.Path,
			string(os.PathListSeparator),
		)

		dotnetCoreAspnetRuntimeLayer.LaunchEnv.Default(
			"DOTNET_ROOT",
			dotnetCoreAspnetRuntimeLayer.Path,
		)

		logger.EnvironmentVariables(dotnetCoreAspnetRuntimeLayer)

		logger.GeneratingSBOM(dotnetCoreAspnetRuntimeLayer.Path)
		var sbomContent sbom.SBOM
		duration, err = clock.Measure(func() error {
			sbomContent, err = sbomGenerator.GenerateFromDependency(dependency, dotnetCoreAspnetRuntimeLayer.Path)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)
		dotnetCoreAspnetRuntimeLayer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		return packit.BuildResult{
			Layers: []packit.Layer{dotnetCoreAspnetRuntimeLayer},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}
