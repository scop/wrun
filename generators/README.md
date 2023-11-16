# wrun command line argument generators

This directory contains scripts to generate wrun command line
arguments for various tools' releases. Generated arguments will
contain `-url`s, and `-archive-exe-path`s if applicable.

The scripts are not robust against all kinds of changes that might be
occurring in upstream release assets, and may need tweaking at times.

The intent is that they should work with the latest respective tool
release, and only that. Generators working with older versions might
be found in wrun Git history.

## Usage

The general usage is:

```shell
python3 wrun_TOOL_args.py VERSION
```

...where `TOOL` is the tool in question, and `VERSION` is the version
to generate for, typically the Git tag rather than the numeric
version if they differ.

The output is newline separated for readability.
Hint: if embedding to a YAML document as a string, e.g. a CI config,
using [line folding (`>-`)](https://yaml.org/spec/1.2.2/#65-line-folding)
the readability can likely be preserved there, too.

## TODO

Some tools for which generators would be nice to have, contributions welcome!

- trivy, https://trivy.dev, https://github.com/aquasecurity/trivy/releases
- typos, https://github.com/crate-ci/typos, https://github.com/crate-ci/typos

## Non-TODO

Some tools for which generators would be nice to have, but cannot be done,
at least yet at time of writing.

Unless mentioned otherwise, reasoning is that there is no `wrun`
runnable asset available for the tool.

- commitlint, https://commitlint.js.org
- gitlint, https://jorisroovers.com/gitlint/
- perlcritic, https://github.com/Perl-Critic/Perl-Critic
- perltidy, https://github.com/perltidy/perltidy
- prettier, https://prettier.io
