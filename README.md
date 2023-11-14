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
falling back to determining based on the digest length,

The first non-flag argument or -- terminates wrun arguments.
Remaining ones are passed to the downloaded one.

Environment variables:
- WRUN_CACHE_HOME: location of the cache, defaults to wrun in the user cache dir
- WRUN_VERBOSE: controls output verbosity; false decreases, true increases
```

## CI usage

Cache resides by default in the
[`$XDG_CACHE_HOME`](https://github.com/adrg/xdg#xdg-base-directory)`/wrun`
directory. `$WRUN_CACHE_HOME` overrides it.

[pre-commit.ci](https://pre-commit.ci) is not supported, because it
[disallows network access at runtime](https://github.com/pre-commit-ci/issues/issues/196#issuecomment-1810937079).

## License

SPDX-License-Identifier: Apache-2.0
