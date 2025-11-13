package main

import (
	"github.com/paketo-buildpacks/dotnet-core-aspnet-runtime/dependency/retrieval/components"
	"github.com/paketo-buildpacks/libdependency/retrieve"
)

func main() {
	fetcher := components.NewFetcher()
	retrieve.NewMetadataWithPlatforms("dotnet-core-aspnet-runtime", fetcher.Get, components.GenerateMetadata)
}
