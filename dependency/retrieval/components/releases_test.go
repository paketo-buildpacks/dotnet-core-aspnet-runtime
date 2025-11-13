package components_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/dotnet-core-aspnet-runtime/dependency/retrieval/components"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testReleases(t *testing.T, context spec.G, it spec.S) {

	var (
		Expect = NewWithT(t).Expect
	)

	context("GetReleases", func() {
		var (
			fetcher components.Fetcher

			releaseIndex *httptest.Server
			releasePage  *httptest.Server
		)

		it.Before(func() {
			releasePage = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodHead {
					http.Error(w, "NotFound", http.StatusNotFound)
					return
				}

				switch req.URL.Path {
				case "/6.0":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
	"eol-date": "2024-11-12",
	"releases": [{
		"aspnetcore-runtime": {
			"version": "6.0.10",
			"files": [{
				"name": "aspnetcore-runtime-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/eb049d47-1cd1-4a76-8b4c-3efee9890f2a/53441bce40b9ac8d073fb4742d823c3b/aspnetcore-runtime-6.0.10-linux-arm.tar.gz",
				"hash": "48d590741a8d648c20e130d3934e6e4a8a4d7ce750c7c74cf4eac77fe969798c36d8780c006baa1514e0b341d3e3cd5a6a3860f484762fc703577d35b1b92202"
      }]
		}
	},
  {
		"aspnetcore-runtime": {
			"version": "6.0.9",
			"files": [{
				"name": "aspnetcore-runtime-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/eb46a420-96cb-4600-95b4-40496349fdf8/f33af6a90cc721adca490d69fa9d0e98/aspnetcore-runtime-6.0.9-linux-arm.tar.gz",
				"hash": "c301b948d5121b4363c8ee9df2915c6c4d588fc0969cae2761f20fb8770bf93e2807b307acca3e313e41adee3f426c47af800b0394700564a480740bd12aa746"
      }]
		}
	}
	]
}`)
				case "/3.1":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
	"eol-date": "2022-12-13",
	"releases": [{
		"aspnetcore-runtime": {
			"version": "3.1.30",
			"files": [{
				"name": "aspnetcore-runtime-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/2cb5afcb-d69c-418a-9be9-661a87aeeed5/bbdf5386457ebac78b97294c74de694e/aspnetcore-runtime-3.1.30-linux-arm.tar.gz",
				"hash": "33e3a6b2e5cffc019a25c4d580047bbf6e927e71da62e043984e76dc4d17d76dfbf8a1576d741038c3bd16ecd6f09c395c08128d85a69adf4cc46a5f803d2853"
      }]
		}
	}
	]
}`)

				case "/non-200":
					w.WriteHeader(http.StatusTeapot)

				case "/no-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `???`)

				case "/no-version-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
	"eol-date": "2022-12-13",
	"releases": [{
		"aspnetcore-runtime": {
			"version": "invalid version",
			"files": [{
				"name": "aspnetcore-runtime-linux-arm.tar.gz",
				"rid": "linux-arm",
				"url": "https://download.visualstudio.microsoft.com/download/pr/2cb5afcb-d69c-418a-9be9-661a87aeeed5/bbdf5386457ebac78b97294c74de694e/aspnetcore-runtime-3.1.30-linux-arm.tar.gz",
				"hash": "33e3a6b2e5cffc019a25c4d580047bbf6e927e71da62e043984e76dc4d17d76dfbf8a1576d741038c3bd16ecd6f09c395c08128d85a69adf4cc46a5f803d2853"
			}]
		}
	}
	]
}`)

				default:
					t.Fatalf("unknown path: %s", req.URL.Path)
				}
			}))

			releaseIndex = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodHead {
					http.Error(w, "NotFound", http.StatusNotFound)
					return
				}

				switch req.URL.Path {
				case "/":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
        {
            "releases.json": "%[1]s/6.0"
        },
				{
            "releases.json": "%[1]s/3.1"
				}
    ]
}\n`, releasePage.URL)

				case "/index-non-200":
					w.WriteHeader(http.StatusTeapot)

				case "/index-no-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `???`)

				case "/release-get-fails":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{
    "releases-index": [
				{
            "releases.json": "Not a valid URL"
				}
    ]
}`)

				case "/release-non-200":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
				{
            "releases.json": "%s/non-200"
				}
    ]
}\n`, releasePage.URL)

				case "/release-no-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
				{
            "releases.json": "%s/no-parse"
				}
    ]
}\n`, releasePage.URL)

				case "/release-no-version-parse":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{
    "releases-index": [
				{
            "releases.json": "%s/no-version-parse"
				}
    ]
}\n`, releasePage.URL)

				default:
					t.Fatalf("unknown path: %s", req.URL.Path)
				}
			}))

			fetcher = components.NewFetcher().WithReleaseIndex(releaseIndex.URL)
		})

		it("fetches a list of relevant releases", func() {
			releases, err := fetcher.Get()
			Expect(err).NotTo(HaveOccurred())

			Expect(releases).To(BeEquivalentTo([]versionology.VersionFetcher{
				components.RuntimeRelease{
					SemVer:         semver.MustParse("6.0.10"),
					EOLDate:        "2024-11-12",
					ReleaseVersion: "6.0.10",
					Files: []components.ReleaseFile{
						{
							Name: "aspnetcore-runtime-linux-arm.tar.gz",
							Rid:  "linux-arm",
							URL:  "https://download.visualstudio.microsoft.com/download/pr/eb049d47-1cd1-4a76-8b4c-3efee9890f2a/53441bce40b9ac8d073fb4742d823c3b/aspnetcore-runtime-6.0.10-linux-arm.tar.gz",
							Hash: "48d590741a8d648c20e130d3934e6e4a8a4d7ce750c7c74cf4eac77fe969798c36d8780c006baa1514e0b341d3e3cd5a6a3860f484762fc703577d35b1b92202",
						},
					},
				},
				components.RuntimeRelease{
					SemVer:         semver.MustParse("6.0.9"),
					EOLDate:        "2024-11-12",
					ReleaseVersion: "6.0.9",
					Files: []components.ReleaseFile{
						{
							Name: "aspnetcore-runtime-linux-arm.tar.gz",
							Rid:  "linux-arm",
							URL:  "https://download.visualstudio.microsoft.com/download/pr/eb46a420-96cb-4600-95b4-40496349fdf8/f33af6a90cc721adca490d69fa9d0e98/aspnetcore-runtime-6.0.9-linux-arm.tar.gz",
							Hash: "c301b948d5121b4363c8ee9df2915c6c4d588fc0969cae2761f20fb8770bf93e2807b307acca3e313e41adee3f426c47af800b0394700564a480740bd12aa746",
						},
					},
				},
				components.RuntimeRelease{
					SemVer:         semver.MustParse("3.1.30"),
					EOLDate:        "2022-12-13",
					ReleaseVersion: "3.1.30",
					Files: []components.ReleaseFile{
						{
							Name: "aspnetcore-runtime-linux-arm.tar.gz",
							Rid:  "linux-arm",
							URL:  "https://download.visualstudio.microsoft.com/download/pr/2cb5afcb-d69c-418a-9be9-661a87aeeed5/bbdf5386457ebac78b97294c74de694e/aspnetcore-runtime-3.1.30-linux-arm.tar.gz",
							Hash: "33e3a6b2e5cffc019a25c4d580047bbf6e927e71da62e043984e76dc4d17d76dfbf8a1576d741038c3bd16ecd6f09c395c08128d85a69adf4cc46a5f803d2853",
						},
					},
				},
			}))
		})

		context("failure cases", func() {
			context("when the index page get fails", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex("not a valid URL")
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
				})
			})

			context("when the index page returns non 200 code", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/index-non-200", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(fmt.Sprintf("received a non 200 status code from %s: status code 418 received", fmt.Sprintf("%s/index-non-200", releaseIndex.URL))))
				})
			})

			context("when the index page cannot be parsed", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/index-no-parse", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("invalid character '?' looking for beginning of value")))
				})
			})

			context("when the release page get fails", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-get-fails", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
				})
			})

			context("when the release page non 200 code", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-non-200", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(fmt.Sprintf("received a non 200 status code from %s: status code 418 received", fmt.Sprintf("%s/non-200", releasePage.URL))))
				})
			})

			context("when the release page cannot be parsed", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-no-parse", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("invalid character '?' looking for beginning of value")))
				})
			})

			context("when the release page has unparsable version", func() {
				it.Before(func() {
					fetcher = fetcher.WithReleaseIndex(fmt.Sprintf("%s/release-no-version-parse", releaseIndex.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("invalid semantic version")))
				})
			})
		})
	})
}
