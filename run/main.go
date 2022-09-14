package main

import (
	"fmt"
	"os"

	"github.com/Netflix/go-env"
	dotnetcoreaspnetruntime "github.com/paketo-buildpacks/dotnet-core-aspnet-runtime"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type Generator struct{}

func (f Generator) GenerateFromDependency(dependency postal.Dependency, path string) (sbom.SBOM, error) {
	return sbom.GenerateFromDependency(dependency, path)
}

func main() {
	var environment dotnetcoreaspnetruntime.Environment
	es, err := env.EnvironToEnvSet(os.Environ())
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to parse build configuration: %w", err))
		os.Exit(1)
	}

	err = env.Unmarshal(es, &environment)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to parse build configuration: %w", err))
		os.Exit(1)
	}

	logEmitter := scribe.NewEmitter(os.Stdout)
	entryResolver := draft.NewPlanner()
	dependencyManager := postal.NewService(cargo.NewTransport())
	runtimeVersionResolver := dotnetcoreaspnetruntime.NewRuntimeVersionResolver(logEmitter, environment)
	runtimeConfigParser := dotnetcoreaspnetruntime.NewRuntimeConfigParser()

	packit.Run(
		dotnetcoreaspnetruntime.Detect(
			environment,
		),
		dotnetcoreaspnetruntime.Build(
			entryResolver,
			dependencyManager,
			runtimeVersionResolver,
			Generator{},
			runtimeConfigParser,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
