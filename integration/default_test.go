package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testDefault(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a container with dotnet-aspnet-runtime", func() {
		var (
			image     occam.Image
			container occam.Container

			name    string
			source  string
			sbomDir string

			err error
		)

		it.Before(func() {
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			sbomDir, err = os.MkdirTemp("", "sbom")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

			Expect(os.RemoveAll(source)).To(Succeed())
			Expect(os.RemoveAll(sbomDir)).To(Succeed())
		})

		it("installs the default dotnet aspnet runtime version into a layer", func() {
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithSBOMOutputDir(sbomDir).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
				"  Resolving ASP.NET Core Runtime version",
				"    Candidate version sources (in priority order):",
				`      runtimeconfig.json -> "6.0.0"`,
				`      <unknown>          -> ""`,
				"",
				"    No exact version match found; attempting version roll-forward",
				"",
				MatchRegexp(`    Selected ASP.NET Core Runtime version \(using runtimeconfig.json\): 6\.0\.\d+`),
				"",
				"  Executing build process",
				MatchRegexp(`    Installing ASP.NET Core Runtime 6\.0\.\d+`),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
				"  Configuring launch environment",
				fmt.Sprintf(`    DOTNET_ROOT -> "/layers/%s/dotnet-core-aspnet-runtime"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				fmt.Sprintf(`    PATH        -> "/layers/%s/dotnet-core-aspnet-runtime:$PATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))

			container, err = docker.Container.Run.
				WithCommand("dotnet --info").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(ContainSubstring(".NET runtimes installed"))

			// check an SBOM file
			contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", "sbom.legacy.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(`"name":"ASP.NET Core Runtime"`))

			// check that all required SBOM files are present
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-aspnet-runtime", "sbom.cdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-aspnet-runtime", "sbom.spdx.json")).To(BeARegularFile())
			Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-aspnet-runtime", "sbom.syft.json")).To(BeARegularFile())

			// check an SBOM file
			contents, err = os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "dotnet-core-aspnet-runtime", "sbom.cdx.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(`"name": "ASP.NET Core Runtime"`))
		})
	})
}
