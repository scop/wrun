# wrun

`wrun` downloads, caches, and runs an executable,
with the same one command for multiple OS/architectures.

```shellsession
$ wrun --help
Usage of wrun:
  -http-timeout duration
    	HTTP client timeout (default 5m0s)
  -url value
    	[<OS>/<architecture>=]URL (at least one required to match)

wrun downloads, caches, and runs executables.
The same one command works for multiple OS/architectures.

The runtime OS and architecture are matched against the given URL matchers.
The first matching one in the order given is chosen as the URL to download.
The matcher OS and architecture may be globs.
As a special case, a plain URL with no matcher part is treated as if it was given with the matcher */*.
URL fragments are treated as hex encoded digests for the download, and checked.
Digest types are identified by type=hexHash or type-hexHash formatted fragments,
falling back to determining based on the digest length.

The first non-flag argument or -- terminates wrun arguments.
Remaining ones are passed to the downloaded one.

Environment variables:
- WRUN_CACHE_HOME: location of the cache, defaults to wrun in the user cache dir
- WRUN_VERBOSE: controls output verbosity; false decreases, true increases
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
Supported fragment formats are `digestAlgo-hexDigest`, `digestAlgo=hexDigest`,
or plain `hexDigest` alone (in which case the digest length is used to determine
the digest algorithm).

For example:

- `#sha256-2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9842`
- `#sha512=9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043`
- `#5d41402abc4b2a76b9719d911017c592` (MD-5 based on length)

## CI usage

Cache resides by default in the
[`$XDG_CACHE_HOME`](https://github.com/adrg/xdg#xdg-base-directory)`/wrun`
directory. `$WRUN_CACHE_HOME` overrides it.

[pre-commit.ci](https://pre-commit.ci) is not supported, because it
[disallows network access at runtime](https://github.com/pre-commit-ci/issues/issues/196#issuecomment-1810937079).

## License

SPDX-License-Identifier: Apache-2.0
