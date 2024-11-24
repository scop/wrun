# wrun

`wrun` downloads, caches, and runs an executable,
with the same one command for multiple OS/architectures.

The primary use case for what it was created is to be able to use a single static command line to download
and invoke executables in git pre-commit hooks, without OS or architecture conditionals.

Executables to download can be standalone as-is, or inside archives.
Downloads are cached locally, and optionally [checksum verified](#download-digests) on download.

<details>
<summary>Detailed usage message</summary>

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
- WRUN_ARGS_FILE: path to file containing command line arguments to prepend, one per line
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

</details>

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

## Usage with [lefthook](https://github.com/evilmartians/lefthook)

See [`.lefthook.yaml` in this repo](.lefthook.yaml) for an example.

Note that because in this repository we want to dogfood the latest wrun itself,
we invoke `go run .` instead of `wrun`.

Another example is in
[scop/vault-token-helper-secret-tool](https://github.com/scop/vault-token-helper-secret-tool).
See [its `.lefthook.yaml`](https://github.com/scop/vault-token-helper-secret-tool/blob/main/.lefthook.yaml)
for how it runs wrun in the lefthook managed git pre-commit hook, and its
[`.github/workflows/check.yml`](https://github.com/scop/vault-token-helper-secret-tool/blob/main/.github/workflows/check.yaml)
for how it installs wrun and installs and runs lefthook in CI.

## Usage with [pre-commit](https://pre-commit.com)

See `.pre-commit-config.yaml` examples in
[`.pre-commit-hooks.yaml`](.pre-commit-hooks.yaml).

[pre-commit.ci](https://pre-commit.ci) is not supported, because it
[disallows network access at runtime](https://github.com/pre-commit-ci/issues/issues/196#issuecomment-1810937079).

## Caching

Cache resides by default in the `wrun` subdirectory of
the [user's cache directory](https://pkg.go.dev/os#UserCacheDir).
`$WRUN_CACHE_HOME` overrides it.

Cache the cache dir in CI to avoid unnecessary executable downloads.
A GitHub actions example is in [this repository's workflow configs](https://github.com/scop/wrun/blob/9438206aac358acf9f13fc8c72cf8297272dfcd3/.github/workflows/check.yaml#L14-L19).

## Generating command line arguments

The `generate` subcommand can be used to generate wrun command line arguments for various tools.

It supports tools shipped in GitHub releases and PyPI executable wrapper wheels that meet its expectations
about asset filenames regarding their OS and architecture.

Some additional tool specific generators are available as well for tools that are not served by the generic GitHub and PyPI generators.
See `wrun generate --help` for more information.

### Automating executable updates

By default, `generate` generates command line arguments pointing to the version of the executable in question that it considers the latest.
This mode together with wrun's ability to load command line arguments from a file
can be used to help with automating executable updates.

An example of this is in this repo's [`.lefthook.yaml`](.lefthook.yaml) (lefthook part),
and [`.github/workflows/updates-tools.html`](.github/workflows/updates-tools.html) (CI part).
[#98](https://github.com/scop/wrun/pull/98) is an example automated pull request automatically created by this config.
(See the note about `go run .` vs `wrun` in the [lefthook](#usage-with-lefthook) chapter above.)

The PR creation part of that makes use of the [peter-evans/create-pull-request](https://github.com/peter-evans/create-pull-request)
GitHub action, but naturally the `generate` subcommand could be run manully and its output copy pasted to appropriate configuration file,
and the related PR created manually.

[Renovate](https://docs.renovatebot.com) e.g. along with its [regex](https://docs.renovatebot.com/modules/manager/regex/) manager
could be used to achieve the same. However, at time of writing (2024-12), [download digests](#download-digests) are not doable with it,
and thus this approach, while possibly more convenient, is arguably inferior compared to the above.

## License

SPDX-License-Identifier: Apache-2.0
