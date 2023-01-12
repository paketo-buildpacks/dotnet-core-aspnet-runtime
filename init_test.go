package dotnetcoreaspnetruntime_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDotnetCoreAspnetRuntime(t *testing.T) {
	suite := spec.New("dotnet-core-aspnet-runtime", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Build", testBuild)
	suite("BuildpackYMLParser", testBuildpackYMLParser)
	suite("Detect", testDetect)
	suite("RuntimeVersionResolver", testRuntimeVersionResolver)
	suite("RuntimeConfigParser", testRuntimeConfigParser)
	suite.Run(t)
}
