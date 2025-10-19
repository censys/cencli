# Censys CLI

`cencli` brings the authority of internet intelligence to your terminal. Analyze assets, perform queries, and hunt threats, all from the command line.

![cencli](examples/cencli.gif)

The Censys CLI is available for free and paid users of the Censys Platform, with support for macOS, Linux, and Windows.

## Quick Start

Make sure you have `cencli` [installed](#installation) and it is on your `$PATH`.

1. Run this command and follow the prompts to add your Censys Platform personal access token:

    ```bash
    $ censys config auth add
    ```

2. (Optional) Run this command and follow the prompts to add your Censys Platform organization ID:

    ```bash
    $ censys config org-id add
    ```

3. That's it! You can now perform asset lookups, searches, and more with the `censys` command.

    ```bash
    $ censys view 8.8.8.8
    ```

## Installation

This section describes how to get `cencli` on your system.

### Homebrew

macOS and Linux users can install `cencli` using [Homebrew](https://brew.sh/):

```bash
$ brew install censys/tap/cencli
```

At the end of the installation process, `zsh` and `bash` completion scripts will be automatically generated and linked to your shell environment.

> [!WARNING]
> Do NOT try to run `brew install censys`. This is a legacy formula that is no longer maintained and in no way affiliated with `cencli`.

For Windows users (and those who prefer not to use Homebrew), you will need to use different methods, which are described below.

### Downloading the Binary

Stable binaries for different platforms (macOS/Linux/Windows) and architectures (amd64/arm64) are available for download on the [releases page](https://github.com/censys/cencli/releases). After you've downloaded and extracted the binary, make sure you add it to your `$PATH`.

> [!WARNING]
> For macOS users, your system may complain about the executable being untrusted after you try to run it. To bypass this, you can run `xattr -dr com.apple.quarantine /path/to/binary` to remove the quarantine flag. If you prefer to do this through the GUI, go to `Settings > Privacy & Security` and allow the executable to be run.

### Go Install

If you have Go 1.25+ installed, you can install `cencli` using the following command:

```bash
$ go install github.com/censys/cencli/cmd/cencli@latest
# make sure to rename the executable to 'censys'
$ mv "$(go env GOPATH)/bin/cencli" "$(go env GOPATH)/bin/censys"
```

### Build from source

Ensure you have Go 1.25+ installed, and run the following commands:

```bash
$ git clone https://github.com/censys/cencli.git
$ cd cencli
$ make censys # builds the executable to ./bin/censys
$ export PATH=$PATH:$(pwd)/bin
$ censys --help
```

## Usage

`cencli` supports various commands for accessing our platform. Run `censys --help` to see all available commands.

### Configuration

The `config` command allows you to manage your personal access tokens and organization IDs. See the [config command docs](./docs/commands/CONFIG.md) for more details.

### View

The `view` command allows you to fetch information about a particular host, certificate, or web property asset at a particular point in time. See the [view command docs](./docs/commands/VIEW.md) for more details.

![view](examples/view/view.gif)

You can also use the `--short` flag to render output using templates, which can be customized. See the [templating documentation](./docs/commands/VIEW.md#templates) for more details.

### Search

The `search` command allows you to perform Censys Platform searches, either globally or within a collection. See the [search command docs](./docs/commands/SEARCH.md) for more details.

![search](examples/search/search.gif)

### Aggregate

The `aggregate` command allows you to perform aggregate queries, either globally or within a collection. See the [aggregate command docs](./docs/commands/AGGREGATE.md) for more details.

![aggregate](examples/aggregate/aggregate.gif)

### Censeye

The `censeye` command allows you to perform a Censeye scan on a host. See the [censeye command docs](./docs/commands/CENSEYE.md) for more details.

![censeye](examples/censeye/censeye-interactive.gif)

### History

This is a WIP. See the [history command docs](./docs/commands/HISTORY.md) for more details.

### Other Commands

- `$ censys completion <bash|zsh|fish|powershell>`: generates shell completion scripts
- `$ censys version`: prints version information

## License

This project is licensed under the Apache License 2.0.
