package dotnetcoreaspnetruntime_test

import (
	"os"
	"path/filepath"
	"testing"

	dotnetcoreaspnetruntime "github.com/paketo-buildpacks/dotnet-core-aspnet-runtime"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRuntimeConfigParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		parser     dotnetcoreaspnetruntime.RuntimeConfigParser
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{}`), 0600)
		Expect(err).NotTo(HaveOccurred())

		parser = dotnetcoreaspnetruntime.NewRuntimeConfigParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Parse", func() {
		it("parses the runtime config", func() {
			frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(frameworkVersion).To(Equal(""))
		})

		context("when the runtime config includes comments", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
					"runtimeOptions": {
						/*
						Multi line
						Comment
						*/
						"configProperties": {
							"System.GC.Server": true
						}
						// comment here ok?
					}
				}`), 0600)).To(Succeed())
			})

			it("parses the runtime config", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal(""))
			})
		})

		context("when the runtime framework version is specified", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
					"runtimeOptions": {
						"framework": {
							"name": "Microsoft.NETCore.App",
							"version": "2.1.3"
						}
					}
				}`), 0600)).To(Succeed())
			})

			it("returns the runtime version", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal("2.1.3"))
			})
		})

		context("when runtime frameworks are specified", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
  "runtimeOptions": {
    "frameworks": [
      {
        "name": "Microsoft.NETCore.App",
        "version": "2.1.3"
      }
    ]
  }
}`), 0600)).To(Succeed())
			})

			it("returns the runtime version", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal("2.1.3"))
			})
		})

		context("when runtime frameworks include AspNetCore", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
  "runtimeOptions": {
    "frameworks": [
      {
        "name": "Microsoft.NETCore.App",
        "version": "2.1.3"
      },
      {
        "name": "Microsoft.AspNetCore.App",
        "version": "2.1.4"
      }
    ]
  }
}`), 0600)).To(Succeed())
			})

			it("returns the runtime and ASPNET versions", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal("2.1.4"))
			})
		})

		context("when the runtime framework is specified with no version", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
					"runtimeOptions": {
						"framework": {
							"name": "Microsoft.NETCore.App"
						}
					}
				}`), 0600)).To(Succeed())
			})

			it("returns that version", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal("*"))
			})
		})

		context("when the app requires ASP.Net via Microsoft.AspNetCore.App", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
					"runtimeOptions": {
						"framework": {
							"name": "Microsoft.AspNetCore.App",
							"version": "2.1.0"
						}
					}
				}`), 0600)).To(Succeed())
			})

			it("sets runtime and ASP.NET versions to the AspNetCore.App version", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal("2.1.0"))
			})
		})

		context("the runtimeconfig.json does not exist", func() {
			it.Before(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, "some-app.runtimeconfig.json"))).To(Succeed())
			})

			it("returns an empty string", func() {
				frameworkVersion, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(frameworkVersion).To(Equal(""))
			})
		})

		context("failure cases", func() {
			context("when given an invalid glob", func() {
				it("returns an error", func() {
					_, err := parser.Parse("[-]")
					Expect(err).To(MatchError(`syntax error in pattern: "[-]"`))
				})
			})

			context("when there are multiple runtimeconfig.json files", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "other-app.runtimeconfig.json"), []byte(`{}`), 0600)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
					Expect(err).To(MatchError(ContainSubstring("multiple *.runtimeconfig.json files present")))
					Expect(err).To(MatchError(ContainSubstring("some-app.runtimeconfig.json")))
					Expect(err).To(MatchError(ContainSubstring("other-app.runtimeconfig.json")))
				})
			})

			context("the runtimeconfig.json file cannot be minimized", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte("var x = /hello"), 0600)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
					Expect(err).To(MatchError(ContainSubstring("unterminated regular expression literal")))
				})
			})

			context("the runtimeconfig.json file cannot be parsed", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`%%%`), 0600)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
					Expect(err).To(MatchError(ContainSubstring("invalid character")))
				})
			})

			context("when frameworks array is specified in runtimeconfig.json", func() {
				context("when the framework object and frameworks array are both in use", func() {
					it.Before(func() {
						Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
			"runtimeOptions": {
			"framework": {
				"name": "Microsoft.AspNetCore.App",
				"version": "2.1.3"
			},
			"frameworks": [
			{
			"name": "Microsoft.AspNetCore.App",
			"version": "2.1.3"
			}
			]
			}
			}`), 0600)).To(Succeed())
					})

					it("returns an error", func() {
						_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
						Expect(err).To(MatchError(ContainSubstring("malformed runtimeconfig.json: multiple 'Microsoft.AspNetCore.App' frameworks specified")))
					})
				})

				context("when there are multiple NETCore framework entries in the frameworks array", func() {
					it.Before(func() {
						Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
				"runtimeOptions": {
				"frameworks": [
				{
				"name": "Microsoft.NETCore.App",
				"version": "2.1.3"
				},
				{
				"name": "Microsoft.NETCore.App",
				"version": "2.0.0"
				}
				]
				}
				}`), 0600)).To(Succeed())
					})

					it("returns an error", func() {
						_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
						Expect(err).To(MatchError(ContainSubstring("malformed runtimeconfig.json: multiple 'Microsoft.NETCore.App' frameworks specified")))
					})
				})

				context("when there are multiple ASP.NET framework entries in the frameworks array", func() {
					it.Before(func() {
						Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
				"runtimeOptions": {
				"frameworks": [
				{
				"name": "Microsoft.AspNetCore.App",
				"version": "2.1.3"
				},
				{
				"name": "Microsoft.AspNetCore.App",
				"version": "2.0.0"
				}
				]
				}
				}`), 0600)).To(Succeed())
					})

					it("returns an error", func() {
						_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
						Expect(err).To(MatchError(ContainSubstring("malformed runtimeconfig.json: multiple 'Microsoft.AspNetCore.App' frameworks specified")))
					})
				})

				context("when there are multiple NETCore framework entries in the frameworks array", func() {
					it.Before(func() {
						Expect(os.WriteFile(filepath.Join(workingDir, "some-app.runtimeconfig.json"), []byte(`{
				"runtimeOptions": {
				"frameworks": [
				{
				"name": "Microsoft.NETCore.App",
				"version": "2.1.3"
				},
				{
				"name": "Microsoft.NETCore.App",
				"version": "2.0.0"
				}
				]
				}
				}`), 0600)).To(Succeed())
					})

					it("returns an error", func() {
						_, err := parser.Parse(filepath.Join(workingDir, "*.runtimeconfig.json"))
						Expect(err).To(MatchError(ContainSubstring("malformed runtimeconfig.json: multiple 'Microsoft.NETCore.App' frameworks specified")))
					})
				})
			})
		})
	})
}
