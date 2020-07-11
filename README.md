# binq-gh

GitHub flavored wrapper CLI for [progrhyme/binq](https://github.com/progrhyme/binq).

# Install

- Download from [GitHub releases](https://github.com/progrhyme/binq-gh/releases)
- `go get github.com/progrhyme/binq-gh/cmd/binq-gh`

# Requirements

To run this CLI, following software are needed:

- [binq](https://github.com/progrhyme/binq) command is installed

# Usage

```sh
binq-gh path/to/item.json [--log-level LOG_LEVEL] [-y|--yes]
```

Refer to [progrhyme/binq](https://github.com/progrhyme/binq) for Item JSON of `binq`.

You can specify the path of `binq` command by `BINQ_BIN` environment variable.
Otherwise, bare `binq` command will be invoked.

Options:

```
-h, --help               # Show help
-L, --log-level string   # Log level (debug,info,notice,warn,error)
-t, --token string       # GitHub API Token
-v, --version            # Show version
-y, --yes                # Update JSON file without confirmation
```

# License

The MIT License.

Copyright (c) 2020 IKEDA Kiyoshi.
