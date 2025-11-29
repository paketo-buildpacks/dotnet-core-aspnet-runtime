package dotnetcoreaspnetruntime_test

import (
	"bytes"
	"os"
	"testing"

	dotnetcoreaspnetruntime "github.com/paketo-buildpacks/dotnet-core-aspnet-runtime"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		detect     packit.DetectFunc
		buffer     *bytes.Buffer
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		detect = dotnetcoreaspnetruntime.Detect(
			dotnetcoreaspnetruntime.Environment{},
			scribe.NewEmitter(buffer),
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("provides dotnet-core-aspnet-runtime", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{
					Name: "dotnet-core-aspnet-runtime",
				},
			},
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
		}))
	})

	context("when BP_DOTNET_RUNTIME_VERSION is set", func() {
		it.Before(func() {
			detect = dotnetcoreaspnetruntime.Detect(
				dotnetcoreaspnetruntime.Environment{
					DotnetRuntimeVersion: "1.2.3",
				},
				scribe.NewEmitter(buffer))
		})

		it("provides and requires dotnet core runtime", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{
						Name: "dotnet-core-aspnet-runtime",
					},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "dotnet-core-aspnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "BP_DOTNET_RUNTIME_VERSION",
							"version":        "1.2.3",
						},
					},
				},
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
			}))
		})
	})

	context("when BP_DOTNET_FRAMEWORK_VERSION is set", func() {
		it.Before(func() {
			detect = dotnetcoreaspnetruntime.Detect(
				dotnetcoreaspnetruntime.Environment{
					DeprecatedDotnetFrameworkVersion: "1.2.3",
				},
				scribe.NewEmitter(buffer))
		})

		it("provides and requires dotnet core runtime", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{
						Name: "dotnet-core-aspnet-runtime",
					},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "dotnet-core-aspnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
							"version":        "1.2.3",
						},
					},
				},
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
			}))
		})
	})
}
