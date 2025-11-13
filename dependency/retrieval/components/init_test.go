package components_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnit(t *testing.T) {
	suite := spec.New("dotnet-core-aspnet-runtime-retrieval", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Dependency", testDependency)
	suite("Releases", testReleases)
	suite.Run(t)
}
