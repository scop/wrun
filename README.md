# wrun

`wrun` downloads, caches, and runs an executable,
with the same one command for multiple OS/architectures.

```shellsession
$ wrun -help
Usage of wrun:
  -archive-exe-path value
    	[OS/arch=]path to executable within archive matcher (separator always /, implies archive processing)
  -http-timeout duration
    	HTTP client timeout (default 5m0s)
  -url value
    	[OS/arch=]URL matcher (at least one required)
  -use-pre-commit-cache
    	Use pre-commit's cache dir

wrun downloads, caches, and runs executables.

OS and architecture matcher arguments for URLs to download and (if applicable) executables within archives can be used to construct command lines that work across multiple operating systems and architectures.

The OS and architecture wrun was built for are matched against the given matchers.
OS and architecture parts of the matcher may be globs.
Order of the matcher arguments is significant: the first match of each is chosen.

As a special case, a matcher argument with no matcher part is treated as if it was given with the matcher */*.

URL fragments, if present, are treated as hashAlgo-hexDigest strings, and downloads are checked against them.

The first non-flag argument or -- terminates wrun arguments.
Remaining ones are passed to the downloaded one.

Environment variables:
- WRUN_CACHE_HOME: cache location, defaults to wrun subdir in the user cache dir
- WRUN_OS_ARCH: override OS/arch for matching
- WRUN_VERBOSE: output verbosity, false decreases, true increases
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
[Go sources](https://cs.opensource.google/go/go/+/refs/tags/go1.21.4:src/cmd/dist/build.go;l=1689-1743).

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
- `#sha512-9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043`
- `#md5-5d41402abc4b2a76b9719d911017c592`)

## CI usage

Cache resides by default in the
[`$XDG_CACHE_HOME`](https://github.com/adrg/xdg#xdg-base-directory)`/wrun`
directory. `$WRUN_CACHE_HOME` overrides it.

If the `-use-pre-commit-cache` flag is given, a subdirectory in
[pre-commit's cache dir](https://pre-commit.com/#managing-ci-caches) is used instead.

[pre-commit.ci](https://pre-commit.ci) is not supported, because it
[disallows network access at runtime](https://github.com/pre-commit-ci/issues/issues/196#issuecomment-1810937079).

## Examples

The [generators](generators/) directory contains scripts that can be
used to generate command line arguments for various tools commonly
used tools.

## License

SPDX-License-Identifier: Apache-2.0
