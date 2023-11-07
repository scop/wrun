# execdl

`execdl` downloads an executable, caches it, and `exec`s it with arguments.

```shellsession
$ execdl --help
Usage of execdl:
  -http-timeout duration
    	HTTP client timeout (default 5m0s)
  -url value
    	[<OS>/<architecture>=]URL (at least one required to match)

execdl downloads, caches, and executes executables.

The runtime OS and architecture are matched against the given URL matchers.
The first matching one in the order given is chosen as the URL to download.
The matcher OS and architecture may be globs.
As a special case, a plain URL with no matcher part is treated as if it was given with the matcher */*.
URL fragments are treated as hex encoded digests for the download, and checked.

The first non-flag argument or -- terminates execdl arguments.
Remaining ones are passed to the downloaded one.

The EXECDL_VERBOSE environment variable controls output verbosity; false decreases, true increases.
```

## License

SPDX-License-Identifier: Apache-2.0
