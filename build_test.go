package dotnetcoreaspnetruntime_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	dotnetcoreaspnetruntime "github.com/paketo-buildpacks/dotnet-core-aspnet-runtime"
	"github.com/paketo-buildpacks/dotnet-core-aspnet-runtime/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string
		buffer     *bytes.Buffer

		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		versionResolver   *fakes.VersionResolver
		sbomGenerator     *fakes.SBOMGenerator
		configParser      *fakes.ConfigParser

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "dotnet-core-aspnet-runtime",
			Metadata: map[string]interface{}{
				"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
				"version":        "2.5.x",
				"launch":         true,
			},
		}

		entryResolver.MergeLayerTypesCall.Returns.Launch = true

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: paketosbom.BOMMetadata{
					Version: "dotnet-core-aspnet-runtime-dep-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "dotnet-core-aspnet-runtime-dep-sha",
					},
					URI: "dotnet-core-aspnet-runtime-dep-uri",
				},
			},
		}

		versionResolver = &fakes.VersionResolver{}
		versionResolver.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "dotnet-core-aspnet-runtime",
			Version: "2.5.x",
			Name:    ".NET Core ASP.NET Runtime",
			SHA256:  "some-sha",
		}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		configParser = &fakes.ConfigParser{}
		configParser.ParseCall.Returns.String = "some-version"

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		build = dotnetcoreaspnetruntime.Build(entryResolver, dependencyManager, versionResolver, sbomGenerator, configParser, logEmitter, chronos.DefaultClock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs the dotnet aspnet runtime libraries", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			Platform: packit.Platform{Path: "platform"},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "dotnet-core-aspnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
							"version":        "2.5.x",
							"launch":         true,
						},
					},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("dotnet-core-aspnet-runtime"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "dotnet-core-aspnet-runtime")))
		Expect(layer.LaunchEnv).To(Equal(packit.Environment{
			"PATH.prepend": filepath.Join(layersDir, "dotnet-core-aspnet-runtime"),
			"PATH.delim":   ":",
		}))
		Expect(layer.Metadata).To(Equal(map[string]interface{}{
			"dependency-sha": "some-sha",
		}))

		Expect(layer.Build).To(BeFalse())
		Expect(layer.Launch).To(BeTrue())
		Expect(layer.Cache).To(BeFalse())

		Expect(layer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
			},
		}))

		Expect(result.Launch.BOM).To(HaveLen(1))
		launchBOMEntry := result.Launch.BOM[0]
		Expect(launchBOMEntry.Name).To(Equal("dotnet-core-aspnet-runtime"))
		Expect(launchBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
			Version: "dotnet-core-aspnet-runtime-dep-version",
			Checksum: paketosbom.BOMChecksum{
				Algorithm: paketosbom.SHA256,
				Hash:      "dotnet-core-aspnet-runtime-dep-sha",
			},
			URI: "dotnet-core-aspnet-runtime-dep-uri",
		}))

		Expect(configParser.ParseCall.Receives.RuntimeConfigFileGlob).To(Equal(filepath.Join(workingDir, "*.runtimeconfig.json")))

		Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version":        "some-version",
					"version-source": "runtimeconfig.json",
				},
			},
			{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
					"version":        "2.5.x",
					"launch":         true,
				},
			},
		}))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:      "dotnet-core-aspnet-runtime",
				Version: "2.5.x",
				Name:    ".NET Core ASP.NET Runtime",
				SHA256:  "some-sha",
			},
		}))

		Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(versionResolver.ResolveCall.Receives.Entry).To(Equal(entryResolver.ResolveCall.Returns.BuildpackPlanEntry))
		Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-core-aspnet-runtime",
			Version: "2.5.x",
			Name:    ".NET Core ASP.NET Runtime",
			SHA256:  "some-sha",
		}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-aspnet-runtime")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-core-aspnet-runtime",
			Version: "2.5.x",
			Name:    ".NET Core ASP.NET Runtime",
			SHA256:  "some-sha",
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "dotnet-core-aspnet-runtime")))
	})

	context("when there is a dependency cache match", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = false

			err := os.WriteFile(filepath.Join(layersDir, "dotnet-core-aspnet-runtime.toml"), []byte("[metadata]\ndependency-sha = \"some-sha\"\n"), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a result that installs the dotnet aspnet runtime libraries", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "dotnet-core-aspnet-runtime",
							Metadata: map[string]interface{}{
								"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
								"version":        "2.5.x",
								"launch":         true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(configParser.ParseCall.Receives.RuntimeConfigFileGlob).To(Equal(filepath.Join(workingDir, "*.runtimeconfig.json")))

			Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-aspnet-runtime",
					Metadata: map[string]interface{}{
						"version":        "some-version",
						"version-source": "runtimeconfig.json",
					},
				},
				{
					Name: "dotnet-core-aspnet-runtime",
					Metadata: map[string]interface{}{
						"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
						"version":        "2.5.x",
						"launch":         true,
					},
				},
			}))

			Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
				{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "2.5.x",
					Name:    ".NET Core ASP.NET Runtime",
					SHA256:  "some-sha",
				},
			}))

			Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
			Expect(versionResolver.ResolveCall.Receives.Entry).To(Equal(entryResolver.ResolveCall.Returns.BuildpackPlanEntry))
			Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyManager.DeliverCall.CallCount).To(Equal(0))
		})
	})

	context("when there is no framework from the runtimeconfig.json or there is no runtimeconfig.json", func() {
		it.Before(func() {
			configParser.ParseCall.Returns.String = ""
		})

		it("returns an empty build result to no-op build", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "dotnet-core-aspnet-runtime",
							Metadata: map[string]interface{}{
								"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
								"version":        "2.5.x",
								"launch":         true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{}))

			Expect(configParser.ParseCall.Receives.RuntimeConfigFileGlob).To(Equal(filepath.Join(workingDir, "*.runtimeconfig.json")))
		})
	})
}
