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

func testLayerReuse(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect       = NewWithT(t).Expect
		Eventually   = NewWithT(t).Eventually
		pack         occam.Pack
		docker       occam.Docker
		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
		imageIDs = map[string]struct{}{}
		containerIDs = map[string]struct{}{}
	})

	context("when an app is rebuilt with no changes", func() {
		var (
			firstImage      occam.Image
			secondImage     occam.Image
			secondContainer occam.Container
			name            string
			source          string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			for containerID := range containerIDs {
				Expect(docker.Container.Remove.Execute(containerID)).To(Succeed())
			}

			for imageID := range imageIDs {
				Expect(docker.Image.Remove.Execute(imageID)).To(Succeed())
			}

			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("reuses the cached runtime layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			firstImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(2))
			firstImageBuildpackMetadata, err := firstImage.BuildpackForKey(buildpackInfo.Buildpack.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstImageBuildpackMetadata.Layers).To(HaveKey("dotnet-core-aspnet-runtime"))

			// second pack build

			secondImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(2))
			secondImageBuildpackMetadata, err := secondImage.BuildpackForKey(buildpackInfo.Buildpack.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(secondImage.Buildpacks[0].Layers).To(HaveKey("dotnet-core-aspnet-runtime"))

			Expect(logs).To(ContainLines(
				"  Resolving ASP.NET Core Runtime version",
				"    Candidate version sources (in priority order):",
				`      runtimeconfig.json -> "6.0.0"`,
				`      <unknown>          -> ""`,
				"",
				"    No exact version match found; attempting version roll-forward",
				"",
				MatchRegexp(`    Selected ASP.NET Core Runtime version \(using runtimeconfig.json\): 6\.0\.\d+`),
				"",
				MatchRegexp(fmt.Sprintf("  Reusing cached layer /layers/%s/dotnet-core-aspnet-runtime", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				"",
			))

			secondContainer, err = docker.Container.Run.
				WithCommand("dotnet --info").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(ContainSubstring(".NET runtimes installed"))

			Expect(secondImageBuildpackMetadata.Layers["dotnet-core-aspnet-runtime"].SHA).To(Equal(firstImageBuildpackMetadata.Layers["dotnet-core-aspnet-runtime"].SHA))
		})
	})

	context("when an app is rebuilt with changed requirements", func() {
		var (
			firstImage      occam.Image
			secondImage     occam.Image
			secondContainer occam.Container
			name            string
			source          string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			for containerID := range containerIDs {
				Expect(docker.Container.Remove.Execute(containerID)).To(Succeed())
			}

			for imageID := range imageIDs {
				Expect(docker.Image.Remove.Execute(imageID)).To(Succeed())
			}

			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("does not reuse the cached runtime layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			firstImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithEnv(map[string]string{
					"BP_DOTNET_FRAMEWORK_VERSION": "3.*",
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[firstImage.ID] = struct{}{}

			Expect(firstImage.Buildpacks).To(HaveLen(2))
			firstImageBuildpackMetadata, err := firstImage.BuildpackForKey(buildpackInfo.Buildpack.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(firstImageBuildpackMetadata.Layers).To(HaveKey("dotnet-core-aspnet-runtime"))

			// second pack build

			secondImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreAspnetRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithEnv(map[string]string{
					"BP_DOTNET_FRAMEWORK_VERSION": "6.*",
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imageIDs[secondImage.ID] = struct{}{}

			Expect(secondImage.Buildpacks).To(HaveLen(2))
			secondImageBuildpackMetadata, err := secondImage.BuildpackForKey(buildpackInfo.Buildpack.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(secondImage.Buildpacks[0].Layers).To(HaveKey("dotnet-core-aspnet-runtime"))

			Expect(logs).To(ContainLines(
				"  Resolving ASP.NET Core Runtime version",
				"    Candidate version sources (in priority order):",
				`      BP_DOTNET_FRAMEWORK_VERSION -> "6.*"`,
				`      runtimeconfig.json          -> "6.0.0"`,
				`      <unknown>                   -> ""`,
				"",
				MatchRegexp(`    Selected ASP.NET Core Runtime version \(using BP_DOTNET_FRAMEWORK_VERSION\): \d+\.\d+\.\d+`),
			))

			Expect(logs).NotTo(ContainSubstring("Reusing cached layer"))

			secondContainer, err = docker.Container.Run.
				WithCommand("dotnet --info").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerIDs[secondContainer.ID] = struct{}{}

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(secondContainer.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(ContainSubstring(".NET runtimes installed"))

			Expect(secondImageBuildpackMetadata.Layers["dotnet-core-aspnet-runtime"].SHA).NotTo(Equal(firstImageBuildpackMetadata.Layers["dotnet-core-aspnet-runtime"].SHA))
		})
	})
}
