# wrun

`wrun` downloads, caches, and runs an executable,
with the same one command for multiple OS/architectures.

```shellsession
$ wrun --help
wrun downloads, caches, and runs executables.

OS and architecture matcher arguments for URLs to download and (if applicable) executables within archives can be used to construct command lines that work across multiple operating systems and architectures.

The OS and architecture wrun was built for are matched against the given matchers.
OS and architecture parts of the matcher may be globs.
Order of the matcher arguments is significant: the first match of each is chosen.

As a special case, a matcher argument with no matcher part is treated as if it was given with the matcher */*.
On Windows, .exe is automatically appended to any archive exe path resulting from a */ prefixed match.

URL fragments, if present, are treated as hashAlgo-hexDigest strings, and downloads are checked against them.

The first non-flag argument or -- terminates wrun arguments.
Remaining ones are passed to the downloaded executable.

Environment variables:
- WRUN_CACHE_HOME: cache location, defaults to wrun subdir in the user's cache dir
- WRUN_OS_ARCH: override OS/arch for matching
- WRUN_VERBOSE: output verbosity, false decreases, true increases

Usage:
  wrun [flags] -- [executable arguments]
  wrun [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  generate    generate wrun command line arguments for various tools
  help        Help about any command

Flags:
  -p, --archive-exe-path strings   [OS/arch=]path to executable within archive matcher (separator always /, implies archive processing)
  -n, --dry-run                    dry run, skip execution (but do download/set up cache)
  -h, --help                       help for wrun
  -t, --http-timeout duration      HTTP client timeout (default 5m0s)
  -u, --url strings                [OS/arch=]URL matcher (at least one required)
  -v, --version                    version for wrun

Use "wrun [command] --help" for more information about a command.
```

## Installation

Prebuilt binaries are available in
[project releases](https://github.com/scop/wrun/releases),
apt and yum package repositories at
[Packagecloud](https://packagecloud.io/scop/wrun).

Prebuilt binaries are also available from PyPI,
in the [`wrun-py`](https://pypi.org/project/wrun-py/) package,
installable for example with `pip`:

```shell
python3 -m pip install wrun-py
```

To build and install from sources, Go is required.

```
go install github.com/scop/wrun@latest
```

## URL matching

URLs are matched against the Go toolchain wrun was built with using
the `OS/architecture=` prefix given along with the URLs. Valid values
for these come from Go, the list is available by running
`go tool dist list`, or from
[Go sources](https://cs.opensource.google/go/go/+/refs/tags/go1.23.2:src/cmd/dist/build.go;l=1728-1778).

OS and architecture may contain globs. The special case where the
`OS/architecture=` prefix is left out is treated as if `*/*=` was
given.

Order of specifying the URLs is significant; the first matching one
is chosen.

## Download digests

To verify downloads against known good digests, place a digest in the URL
fragment.
The fragment format to use is `digestAlgo-hexDigest`.

For example:

- `#sha256-2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9842`

## Usage with pre-commit

See `.pre-commit-config.yaml` examples in
[`.pre-commit-hooks.yaml`](.pre-commit-hooks.yaml).

## Usage in CI

Cache resides by default in the `wrun` subdirectory of
the [user's cache directory](https://pkg.go.dev/os#UserCacheDir).
`$WRUN_CACHE_HOME` overrides it.

[pre-commit.ci](https://pre-commit.ci) is not supported, because it
[disallows network access at runtime](https://github.com/pre-commit-ci/issues/issues/196#issuecomment-1810937079).

## Generating command line arguments

The `generate` subcommand can be used to generate wrun command line arguments for various tools.

It supports tools shipped in GitHub releases and PyPI executable wrapper wheels that meet its expectations
about asset filenames regarding their OS and architecture.

Some additional tool specific generators are available as well for tools that are not served by the generic GitHub and PyPI generators.
See `wrun generate --help` for more information.

## License

SPDX-License-Identifier: Apache-2.0
