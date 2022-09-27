package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/postal"
)

type VersionResolver struct {
	ResolveCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Path  string
			Entry packit.BuildpackPlanEntry
			Stack string
		}
		Returns struct {
			Dependency postal.Dependency
			Error      error
		}
		Stub func(string, packit.BuildpackPlanEntry, string) (postal.Dependency, error)
	}
}

func (f *VersionResolver) Resolve(param1 string, param2 packit.BuildpackPlanEntry, param3 string) (postal.Dependency, error) {
	f.ResolveCall.mutex.Lock()
	defer f.ResolveCall.mutex.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.Path = param1
	f.ResolveCall.Receives.Entry = param2
	f.ResolveCall.Receives.Stack = param3
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2, param3)
	}
	return f.ResolveCall.Returns.Dependency, f.ResolveCall.Returns.Error
}
