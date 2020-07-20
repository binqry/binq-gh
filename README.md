# binq-gh

GitHub flavored wrapper CLI for [binqry/binq](https://github.com/binqry/binq).

This tool fetches latest GitHub release of a binq item and update the item JSON file.

# Install

- Homebrew (or Linuxbrew) using Tap [progrhyme/tap](https://github.com/progrhyme/homebrew-tap)
- Download from [GitHub releases](https://github.com/binqry/binq-gh/releases)
- `go get github.com/binqry/binq-gh/cmd/binq-gh`

# Requirements

To run this CLI, following software are needed:

- [binq](https://github.com/binqry/binq) command is installed

# Usage

```sh
# Check & update Item JSON of binq
binq-gh path/to/item.json [-t|--token GITHUB_TOKEN] \
  [-L|--log-level LOG_LEVEL] [-y|--yes]
```

Refer to [binqry/binq](https://github.com/binqry/binq) for Item JSON of `binq`.

If the URL format in item JSON does not match https://github.com/, `binq-gh` does nothing.

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
