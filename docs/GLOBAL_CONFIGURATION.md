# Global Configuration

On first run, `cencli` creates a configuration directory at `~/.config/cencli` (or `$CENCLI_DATA_DIR` if set) containing:

- `config.yaml` - Global configuration file with default settings
- `cencli.db` - SQLite database for storing authentication credentials and other persistent data
- `templates/` - Directory containing Handlebars templates for formatted output

## Configuration File

The `config.yaml` file is automatically generated with sensible defaults. All configuration values can be overridden via command-line flags or environment variables.

### Configuration Precedence

Configuration values are resolved in the following order (highest to lowest priority):

1. Command-line flags
2. Environment variables (prefixed with `CENCLI_`)
3. Configuration file (`config.yaml`)
4. Default values

## Global Flags

### `--output-format`, `-O`

Default output format for command results.

**Flag:** `--output-format`, `-O`  
**Environment Variable:** `CENCLI_OUTPUT_FORMAT`  
**Type:** `string`  
**Default:** `json` (globally), but individual commands may default to `short`  
**Valid Values:** `json`, `yaml`, `tree`, `short`, `template`

Controls how data is formatted when printed to stdout:

- **`json`** - Structured JSON output (default for most commands)
- **`yaml`** - Structured YAML output
- **`tree`** - Hierarchical tree view of nested data structures
- **`short`** - Human-readable formatted output (available on select commands like `aggregate`, `censeye`, `search`, `view`)
- **`template`** - Render using custom Handlebars templates (available on `search` and `view` commands)

**Note:** Some commands default to `short` output instead of `json` to provide a better user experience. For example, the `aggregate` and `censeye` commands show formatted tables by default. You can always override this with `--output-format json` or another format.

### `--streaming`, `-S`

Enable streaming output mode.

**Flag:** `--streaming`, `-S`  
**Environment Variable:** `CENCLI_STREAMING`  
**Type:** `boolean`  
**Default:** `false`

When enabled, commands that support streaming will output results as NDJSON (newline-delimited JSON) with each record emitted immediately as data is fetched. This provides several benefits for large result sets:

- **Output begins before all pages are fetched** - You see results as soon as the first page is retrieved
- **Memory usage stays bounded** - Results are written immediately rather than accumulated in memory
- **Partial results are preserved on interruption** - If you press Ctrl-C or an error occurs, all previously emitted records remain intact
- **Safe for large queries** - Ideal for use with `--max-pages -1` when fetching potentially unbounded result sets

**Supported commands:** `search`, `view`, `history`

**Note:** `--streaming` cannot be used together with `--output-format`. When streaming mode is enabled, output is always NDJSON. If you set `streaming: true` in your config file, it will be silently ignored for commands that don't support streaming.

### `--no-color`

Disable ANSI colors and styles in output.

**Flag:** `--no-color`  
**Environment Variable:** `CENCLI_NO_COLOR`  
**Type:** `boolean`  
**Default:** `false`

When enabled, all output will be rendered without color or styling. This is useful for piping output to files or other commands.

### `--no-spinner`

Disable spinner animations during operations.

**Flag:** `--no-spinner`  
**Environment Variable:** `CENCLI_NO_SPINNER`  
**Type:** `boolean`  
**Default:** `false`

Disables the loading spinner that appears during long-running operations. Useful for non-interactive environments or when output is being logged.

### `--quiet`, `-q`

Suppress non-essential output.

**Flag:** `--quiet`, `-q`  
**Environment Variable:** `CENCLI_QUIET`  
**Type:** `boolean`  
**Default:** `false`

When enabled, suppresses response metadata and other informational messages, showing only the primary command output.

### `--debug`

Enable debug logging.

**Flag:** `--debug`  
**Environment Variable:** `CENCLI_DEBUG`  
**Type:** `boolean`  
**Default:** `false`

Enables verbose debug logging, including HTTP requests, response details, and internal state information. Useful for troubleshooting issues.

### `timeouts.http`

Overall command timeout.

**Flag:** `--timeout-http`  
**Environment Variable:** `CENCLI_TIMEOUTS_HTTP`  
**Type:** `duration`  
**Default:** `0`

Sets the maximum time an individual HTTP request can take before timing out. Accepts duration strings like `30s`, `2m`, `1h30m`. Set to `0` to disable.

## Spinner

The spinner configuration controls the spinner UI.

### `spinner.disabled`

Disable the spinner.

**Flag:** `--no-spinner`  
**Environment Variable:** `CENCLI_SPINNER_DISABLED`  
**Type:** `boolean`  
**Default:** `false`

### `spinner.start-stopwatch-after`

Show stopwatch in the spinner after this many seconds.

**Environment Variable:** `CENCLI_SPINNER_START_STOPWATCH_AFTER`  
**Type:** `integer`  
**Default:** `5`

## Retry Strategy

The retry strategy configuration controls how the CLI handles failed API requests.

### `retry-strategy.max-attempts`

Maximum number of retry attempts for failed requests.

**Environment Variable:** `CENCLI_RETRY_STRATEGY_MAX_ATTEMPTS`  
**Type:** `integer`  
**Default:** `2`

### `retry-strategy.base-delay`

Initial delay between retry attempts.

**Environment Variable:** `CENCLI_RETRY_STRATEGY_BASE_DELAY`  
**Type:** `duration`  
**Default:** `500ms`

### `retry-strategy.max-delay`

Maximum delay between retry attempts.

**Environment Variable:** `CENCLI_RETRY_STRATEGY_MAX_DELAY`  
**Type:** `duration`  
**Default:** `30s`

### `retry-strategy.backoff`

Backoff strategy for calculating retry delays.

**Environment Variable:** `CENCLI_RETRY_STRATEGY_BACKOFF`  
**Type:** `string`  
**Default:** `fixed`  
**Valid Values:** `fixed`, `linear`, `exponential`

- `fixed`: Use the same delay for all retries (base-delay)
- `linear`: Increase delay linearly with each retry
- `exponential`: Double the delay with each retry (exponential backoff)

## Search Configuration

Default settings for the `search` command. Note that these are not bound to global flags and are only applied to the `search` command.

### `search.page-size`

Default number of results per page for search operations.

**Environment Variable:** `CENCLI_SEARCH_PAGE_SIZE`  
**Type:** `integer`  
**Default:** `100`  
**Constraints:** Must be >= 1

### `search.max-pages`

Maximum number of pages to fetch during search operations.

**Environment Variable:** `CENCLI_SEARCH_MAX_PAGES`  
**Type:** `integer`  
**Default:** `1`  
**Constraints:** Set to `-1` for unlimited (up to API maximum of 100 pages)

## Default Timezone

The default timezone used for parsing timestamp inputs that don't include timezone information.

### `default-tz`

Default timezone for interpreting timestamps without explicit timezone information.

**Environment Variable:** `CENCLI_DEFAULT_TZ`  
**Type:** `string` (timezone identifier)  
**Default:** `UTC`

When you provide timestamps without timezone information (like `2025-09-15 14:30:00` or `2025-09-15`), `cencli` interprets them using this configured timezone. Check out the [view command docs](commands/VIEW.md#timestamps) for details on timestamp format support and usage examples.

For the complete, authoritative list of supported timezones, see [timezones.go](../internal/pkg/datetime/timezones.go). If you need a timezone that isn't listed, please open an issue or submit a pull request.

## Templates

Template paths for formatted output. Each asset type has its own template file. See [the view command docs](commands/VIEW.md#templates) for more details.

## Standard Environment Variables

In addition to `cencli`-specific environment variables, the CLI respects the following standard environment variables:

### `NO_COLOR`

Disable colored output (follows the [NO_COLOR](https://no-color.org/) standard).

**Type:** `boolean` (`1` or `true`)  
**Default:** not set

When set to `1` or `true`, disables all ANSI colors and styles in output. This takes precedence over the `--no-color` flag and `CENCLI_NO_COLOR` environment variable.

```bash
NO_COLOR=1 censys view 8.8.8.8
```

### `FORCE_COLOR`

Force colored output even when not connected to a TTY (follows the [FORCE_COLOR](https://force-color.org/) standard).

**Type:** `boolean` (`1` or `true`)  
**Default:** not set

When set to `1` or `true`, forces colored output even when stdout is not a terminal. Useful when piping output through tools that preserve ANSI codes.

```bash
FORCE_COLOR=1 censys view 8.8.8.8 | less -R
```

### `CENCLI_DATA_DIR`

Override the default configuration directory location.

**Type:** `string` (directory path)  
**Default:** `~/.config/cencli`

When set, `cencli` will use this directory instead of `~/.config/cencli` for storing configuration, templates, and the database.

```bash
CENCLI_DATA_DIR=/custom/path censys config auth add
```
