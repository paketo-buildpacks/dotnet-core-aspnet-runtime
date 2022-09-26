package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testRuntimeconfigNoop(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a container with dotnet-aspnet-runtime and the runtimecofig.json does not specify a framework", func() {
		var (
			image occam.Image

			name   string
			source string

			err error
		)

		it.Before(func() {
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			Expect(os.WriteFile(filepath.Join(source, "fde_dotnet_6.runtimeconfig.json"), []byte(`{
  "runtimeOptions": {
    "configProperties": {
      "System.GC.Server": true,
      "System.Reflection.Metadata.MetadataUpdater.IsSupported": false,
      "System.Runtime.Serialization.EnableUnsafeBinaryFormatterSerialization": false
    }
  }
}`), 0644)).To(Succeed())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("no-ops the build process", func() {
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
				"  Skipping build process. No required runtime frameworks listed in runtimeconfig.json. App is self-contained.",
				"",
			))
		})
	})

	context("when building a container with dotnet-aspnet-runtime and there is no runtimecofig.json", func() {
		var (
			image occam.Image

			name   string
			source string

			err error
		)

		it.Before(func() {
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			Expect(os.Remove(filepath.Join(source, "fde_dotnet_6.runtimeconfig.json"))).To(Succeed())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("no-ops the build process", func() {
			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
				"  Skipping build process. No runtimeconfig.json found.",
				"",
			))
		})
	})
}
