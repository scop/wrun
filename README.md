# wrun

`wrun` downloads an executable, caches it, and runs it with arguments.

```shellsession
$ wrun --help
Usage of wrun:
  -http-timeout duration
    	HTTP client timeout (default 5m0s)
  -url value
    	[<OS>/<architecture>=]URL (at least one required to match)

wrun downloads, caches, and runs executables.

The runtime OS and architecture are matched against the given URL matchers.
The first matching one in the order given is chosen as the URL to download.
The matcher OS and architecture may be globs.
As a special case, a plain URL with no matcher part is treated as if it was given with the matcher */*.
URL fragments are treated as hex encoded digests for the download, and checked.

The first non-flag argument or -- terminates wrun arguments.
Remaining ones are passed to the downloaded one.

Environment variables:
- WRUN_CACHE_HOME: location of the cache, defaults to wrun in the user cache dir
- WRUN_VERBOSE: controls output verbosity; false decreases, true increases
```

## License

SPDX-License-Identifier: Apache-2.0
