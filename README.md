# Paketo Buildpack for ASP.NET Core Runtime

The ASP.NET Core Runtime CNB provides a version of the [ASP.NET Core
Runtime](https://learn.microsoft.com/en-us/aspnet/core/?view=aspnetcore-6.0).
The ASP.NET Core Runtime dependency will be made available on the `$PATH` and
`$DOTNET_ROOT` at run-time for .NET Core apps containing a
`*.runtimeconfig.json` file with runtime frameworks specified.

The buildpack is published for consumption as an image at
`paketobuildpacks/dotnet-core-aspnet-runtime`. It is a part of the [Paketo
Buildpack for .NET Core](https://github.com/paketo-buildpacks/dotnet-core),
which is a top-level language family buildpack that leverages all related .NET
Core buildpacks together.

## Integration

The ASP.NET Core Runtime CNB provides `dotnet-core-aspnet-runtime` as a
dependency. Downstream buildpacks, like [.NET Core
Execute](https://github.com/paketo-buildpacks/dotnet-execute) can require the
`dotnet-core-aspnet-runtime` dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the ASP.NET Core Runtime dependency is "dotnet-core-aspnet-runtime".
  # This value is considered part of the public API for the buildpack and will
  # not change without a plan for deprecation.
  name = "dotnet-core-aspnet-runtime"

  # The ASP.NET Core Runtime buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the launch flag to true will ensure that the ASP.NET Core Runtime
    # dependency is available on the $PATH and $DOTNET_ROOT for the running
    # application. If you are writing an application that needs to run ASP.NET
    # Runtime at runtime, this flag should be set to true.
    launch = true

    # Setting the build flag to true will ensure that the ASP.NET Core Runtime
    # dependency is available to subsequent buildpacks during their build phase.
    # This is NOT recommended, because most .NET Core apps will also need the
    # .NET Core SDK dependency during build-time. The dotnet-core-sdk
    # dependency (provided by the separate .NET Core SDK buildpack) includes the
    # SDK, as well as the ASP.NET Core Runtime and should be used instead.
    build = true

    # The version of the ASP.NET Core Runtime dependency is not required. In the
    # case it is not specified, the buildpack will determine and provide a
    # version based off of the *runtimeconfig.json file. If you wish to request a
    # specific version, the buildpack supports specifying a semver constraint in
    # the form of "6.*", "6.0.*", or even "6.0.5".
    version = "6.0.5"
```

## Configuration

### `BP_DOTNET_FRAMEWORK_VERSION`
The `BP_DOTNET_FRAMEWORK_VERSION` variable allows you to specify the version of
ASP.NET Core Runtime that is installed. The environment variable can be
set at build-time either directly  (ex. `pack build my-app --env
BP_ENVIRONMENT_VARIABLE=some-value`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)

```shell
BP_DOTNET_FRAMEWORK_VERSION=6.0.5
```

### `BP_LOG_LEVEL`
The `BP_LOG_LEVEL` variable allows you to configure the level of log output
from the **buildpack itself**.  The environment variable can be set at build
time either directly (ex. `pack build my-app --env BP_LOG_LEVEL=DEBUG`) or
through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)
If no value is set, the default value of `INFO` will be used.

The options for this setting are:
- `INFO`: (Default) log information about the progress of the build process
- `DEBUG`: log debugging information about the progress of the build process

```shell
BP_LOG_LEVEL="DEBUG"
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
`pack build <app-name> -p <path-to-app> -b build/buildpackage.cnb -b <other-buildpacks..>`

To run the unit and integration tests for this buildpack:
```
$ ./scripts/unit.sh && ./scripts/integration.sh
```
