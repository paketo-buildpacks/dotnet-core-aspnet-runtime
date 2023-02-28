package dotnetcoreaspnetruntime_test

import (
	"bytes"
	"errors"
	"io"
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
			ID:       "dotnet-core-aspnet-runtime",
			Version:  "2.5.x",
			Name:     ".NET Core ASP.NET Runtime",
			Checksum: "sha512:some-sha",
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
			"PATH.prepend":        filepath.Join(layersDir, "dotnet-core-aspnet-runtime"),
			"PATH.delim":          ":",
			"DOTNET_ROOT.default": filepath.Join(layersDir, "dotnet-core-aspnet-runtime"),
		}))
		Expect(layer.Metadata).To(Equal(map[string]interface{}{
			"dependency-checksum": "sha512:some-sha",
		}))

		Expect(layer.Build).To(BeFalse())
		Expect(layer.Launch).To(BeTrue())
		Expect(layer.Cache).To(BeFalse())

		Expect(layer.SBOM.Formats()).To(HaveLen(2))
		cdx := layer.SBOM.Formats()[0]
		spdx := layer.SBOM.Formats()[1]

		Expect(cdx.Extension).To(Equal("cdx.json"))
		content, err := io.ReadAll(cdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"bomFormat": "CycloneDX",
			"components": [],
			"metadata": {
				"tools": [
					{
						"name": "syft",
						"vendor": "anchore",
						"version": "[not provided]"
					}
				]
			},
			"specVersion": "1.3",
			"version": 1
		}`))

		Expect(spdx.Extension).To(Equal("spdx.json"))
		content, err = io.ReadAll(spdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"SPDXID": "SPDXRef-DOCUMENT",
			"creationInfo": {
				"created": "0001-01-01T00:00:00Z",
				"creators": [
					"Organization: Anchore, Inc",
					"Tool: syft-"
				],
				"licenseListVersion": "3.16"
			},
			"dataLicense": "CC0-1.0",
			"documentNamespace": "https://paketo.io/packit/unknown-source-type/unknown-88cfa225-65e0-5755-895f-c1c8f10fde76",
			"name": "unknown",
			"relationships": [
				{
					"relatedSpdxElement": "SPDXRef-DOCUMENT",
					"relationshipType": "DESCRIBES",
					"spdxElementId": "SPDXRef-DOCUMENT"
				}
			],
			"spdxVersion": "SPDX-2.2"
		}`))

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
				ID:       "dotnet-core-aspnet-runtime",
				Version:  "2.5.x",
				Name:     ".NET Core ASP.NET Runtime",
				Checksum: "sha512:some-sha",
			},
		}))

		Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(versionResolver.ResolveCall.Receives.Entry).To(Equal(entryResolver.ResolveCall.Returns.BuildpackPlanEntry))
		Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:       "dotnet-core-aspnet-runtime",
			Version:  "2.5.x",
			Name:     ".NET Core ASP.NET Runtime",
			Checksum: "sha512:some-sha",
		}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-aspnet-runtime")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:       "dotnet-core-aspnet-runtime",
			Version:  "2.5.x",
			Name:     ".NET Core ASP.NET Runtime",
			Checksum: "sha512:some-sha",
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "dotnet-core-aspnet-runtime")))
	})

	context("when there is a dependency cache match", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = false

			err := os.WriteFile(filepath.Join(layersDir, "dotnet-core-aspnet-runtime.toml"), []byte("[metadata]\ndependency-checksum = \"sha512:some-sha\"\n"), 0600)
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
					ID:       "dotnet-core-aspnet-runtime",
					Version:  "2.5.x",
					Name:     ".NET Core ASP.NET Runtime",
					Checksum: "sha512:some-sha",
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

	context("when the backwards compatible api is being used", func() {
		it("returns a result that installs the dotnet aspnet runtime libraries and prints a warning", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:        "Some Buildpack",
					Version:     "1.2.3",
					SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
				},
				Platform: packit.Platform{Path: "platform"},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "dotnet-runtime",
							Metadata: map[string]interface{}{
								"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
								"version":        "2.5.x",
								"launch":         true,
							},
						},
						{
							Name: "dotnet-aspnetcore",
							Metadata: map[string]interface{}{
								"launch": true,
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
				{
					Name: "dotnet-core-aspnet-runtime",
					Metadata: map[string]interface{}{
						"launch": true,
					},
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("WARNING: Requiring dotnet-runtime or dotnet-aspnetcore in your build plan will be deprecated soon in .NET Core Buildpack v2.0.0."))
			Expect(buffer.String()).To(ContainSubstring("Please require dotnet-core-aspnet-runtime in your build plan going forward."))
		})
	})

	context("when version-source of the selected entry is buildpack.yml", func() {
		it.Before(func() {
			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "dotnet-core-aspnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "2.5.x",
					"launch":         true,
				},
			}
		})

		it("chooses the specified version and emits a warning", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "dotnet-core-aspnet-runtime",
							Metadata: map[string]interface{}{
								"version-source": "buildpack.yml",
								"version":        "2.5.x",
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(buffer.String()).To(ContainSubstring("WARNING: Setting the .NET Framework version through buildpack.yml will be deprecated soon in .NET Core ASP.NET Core Runtime Buildpack v2.0.0."))
			Expect(buffer.String()).To(ContainSubstring("Please specify the version through the $BP_DOTNET_FRAMEWORK_VERSION environment variable instead. See docs for more information."))
		})
	})

	context("failure cases", func() {
		context("when the config parser fails", func() {
			it.Before(func() {
				configParser.ParseCall.Returns.Error = errors.New("failed to parse config")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-aspnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("failed to parse config"))
			})
		})

		context("when a dependency cannot be resolved", func() {
			it.Before(func() {
				versionResolver.ResolveCall.Returns.Error = errors.New("failed to resolve version")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-aspnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("failed to resolve version"))
			})
		})

		context("when the layer get fails", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-aspnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when a dependency download fails", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("failed to download dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-aspnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("failed to download dependency"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-aspnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{SBOMFormats: []string{"random-format"}},
					CNBPath:       cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-runtime",
								Metadata: map[string]interface{}{
									"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
			})
		})
	})
}
