package dotnetcoreaspnetruntime_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	dotnetcoreaspnetruntime "github.com/paketo-buildpacks/dotnet-core-aspnet-runtime"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRuntimeVersionResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer          *bytes.Buffer
		logEmitter      scribe.Emitter
		cnbDir          string
		buildpackToml   string
		versionResolver dotnetcoreaspnetruntime.RuntimeVersionResolver
		entry           packit.BuildpackPlanEntry
	)

	it.Before(func() {
		var err error

		buffer = bytes.NewBuffer(nil)
		logEmitter = scribe.NewEmitter(buffer)

		versionResolver = dotnetcoreaspnetruntime.NewRuntimeVersionResolver(logEmitter, dotnetcoreaspnetruntime.Environment{})

		cnbDir, err = os.MkdirTemp("", "cnb")
		buildpackToml = filepath.Join(cnbDir, "buildpack.toml")
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(buildpackToml, []byte(`api = "0.2"
[buildpack]
  id = "org.some-org.some-buildpack"
  name = "Some Buildpack"
  version = "some-version"

[metadata]

	[metadata.default-versions]
		dotnet-core-aspnet-runtime = "1.2.0"
	
  [[metadata.dependencies]]
    id = "dotnet-core-aspnet-runtime"
    sha256 = "some-sha"
    stacks = ["some-stack"]
    uri = "some-uri"
    version = "1.2.2"
	
  [[metadata.dependencies]]
    id = "dotnet-core-aspnet-runtime"
    sha256 = "some-sha"
    stacks = ["some-stack"]
    uri = "some-uri"
    version = "2.2.3"

  [[metadata.dependencies]]
    id = "dotnet-core-aspnet-runtime"
    sha256 = "some-sha"
    stacks = ["some-stack"]
    uri = "some-uri"
    version = "2.2.4"
`), 0600)
		Expect(err).NotTo(HaveOccurred())

		entry = packit.BuildpackPlanEntry{
			Name: "dotnet-core-aspnet-runtime",
			Metadata: map[string]interface{}{
				"version-source": "UNKNOWN",
				"launch":         true,
			},
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	context("the version source is empty", func() {
		it.Before(func() {
			delete(entry.Metadata, "version-source")
			entry.Metadata["version"] = "1.2.0"
		})

		it("returns the default version", func() {
			dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-core-aspnet-runtime",
				Version: "1.2.2",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the version source is runtimeconfig.json", func() {
		it.Before(func() {
			entry.Metadata["version-source"] = "runtimeconfig.json"
		})

		context("the buildpack.toml has the exact version", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.3"
			})
			it("returns a dependency with that version", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "2.2.3",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("the buildpack.toml only has a major minor version match", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.0"
			})
			it("returns a compatible version", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "2.2.4",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("the buildpack.toml only has a major minor version match with BP_DOTNET_ROLL_FORWARD=Disable", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.0"
				versionResolver = dotnetcoreaspnetruntime.NewRuntimeVersionResolver(logEmitter, dotnetcoreaspnetruntime.Environment{
					DotnetRollForward: "Disable",
				})
			})
			it("returns a compatible version", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "dotnet-core-aspnet-runtime" dependency for stack "some-stack" with version constraint "2.2.0": no compatible versions. Supported versions are: [1.2.2, 2.2.3, 2.2.4]. This may be due to BP_DOTNET_ROLL_FORWARD=Disable`)))
			})
		})

		context("the buildpack.toml only has a major version match", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.1.7"
			})
			it("returns a compatible version", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "2.2.4",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("the buildpack.toml does not have a version match", func() {
			context("the requested version is a major version higher", func() {
				it.Before(func() {
					entry.Metadata["version"] = "3.0.0"
				})
				it("returns an error", func() {
					_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
					Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "dotnet-core-aspnet-runtime" dependency for stack "some-stack" with version constraint "3.0.0": no compatible versions. Supported versions are: [1.2.2, 2.2.3, 2.2.4]`)))
					Expect(err).NotTo(MatchError(ContainSubstring(`. This may be due to BP_DOTNET_ROLL_FORWARD=Disable`)))
				})
			})

			context("the requested version is a minor version higher", func() {
				it.Before(func() {
					entry.Metadata["version"] = "2.3.0"
				})
				it("returns an error", func() {
					_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
					Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "dotnet-core-aspnet-runtime" dependency for stack "some-stack" with version constraint "2.3.0": no compatible versions. Supported versions are: [1.2.2, 2.2.3, 2.2.4]`)))
				})
			})

			context("the requested version is a patch version higher", func() {
				it.Before(func() {
					entry.Metadata["version"] = "2.2.5"
				})
				it("returns an error", func() {
					_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
					Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "dotnet-core-aspnet-runtime" dependency for stack "some-stack" with version constraint "2.2.5": no compatible versions. Supported versions are: [1.2.2, 2.2.3, 2.2.4]`)))
				})
			})
		})

		context("the buildpack.toml does not have a dependency with a matching ID", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.3"
				entry.Name = "random-ID"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "random-ID" dependency for stack "some-stack" with version constraint "2.2.3": no compatible versions. Supported versions are: []`)))
			})
		})

		context("the buildpack.toml does not have a dependency with a matching stack", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.3"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "random-stack")
				Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "dotnet-core-aspnet-runtime" dependency for stack "random-stack" with version constraint "2.2.3": no compatible versions. Supported versions are: []`)))
			})
		})

		context("the version has a wildcard patch", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.1.*"
			})
			it("allows patch and minor version rollforward", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "2.2.4",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("the version is empty", func() {
			it("returns the default version", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "1.2.2",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})

		context("the version is default", func() {
			it.Before(func() {
				entry.Metadata["version"] = "default"
			})
			it("returns the default version", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "1.2.2",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})
	})

	context("the version source is BP_DOTNET_FRAMEWORK_VERSION", func() {
		it.Before(func() {
			entry.Metadata["version-source"] = "BP_DOTNET_FRAMEWORK_VERSION"
		})

		context("the buildpack.toml only has a major minor version match", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.0"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring(`failed to satisfy "dotnet-core-aspnet-runtime" dependency for stack "some-stack" with version constraint "2.2.0": no compatible versions. Supported versions are: [1.2.2, 2.2.3, 2.2.4]`)))
			})
		})

		context("the version contains a `*`", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.2.*"
			})
			it("attempts to turn the given versions into the only constraint", func() {
				dependency, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-core-aspnet-runtime",
					Version: "2.2.4",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})
	})

	context("failure cases", func() {
		context("the buildpack.toml cannot be parsed", func() {
			it.Before(func() {
				Expect(os.WriteFile(buildpackToml, []byte(`%%%`), 0600)).To(Succeed())
				entry.Metadata["version"] = "2.2.3"
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("expected '.' or '=', but got '%' instead")))
			})
		})

		context("the version is not semver compatible", func() {
			it.Before(func() {
				entry.Metadata["version"] = "invalid-version"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("improper constraint")))
			})
		})

		context("a buildpack.toml version is not semver compatible", func() {
			it.Before(func() {
				err := os.WriteFile(buildpackToml, []byte(`api = "0.2"
[buildpack]
id = "org.some-org.some-buildpack"
name = "Some Buildpack"
version = "some-version"

[metadata]

[[metadata.dependencies]]
id = "dotnet-core-aspnet-runtime"
sha256 = "some-sha"
stacks = ["some-stack"]
uri = "some-uri"
version = "invalid-version"

[[metadata.dependencies]]
id = "dotnet-core-aspnet-runtime"
sha256 = "some-sha"
stacks = ["some-stack"]
uri = "some-uri"
version = "2.2.4"
`), 0600)
				Expect(err).NotTo(HaveOccurred())
				entry.Metadata["version"] = "1.2.0"
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(buildpackToml, entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("invalid semantic version")))
			})
		})
	})

}
