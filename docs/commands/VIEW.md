# View Command

The `view` command allows you to retrieve detailed information about hosts, certificates, and web properties from the Censys Platform.

![view](../../examples/view/view.gif)

## Usage

```bash
$ censys view 8.8.8.8 # view a single host
$ censys view 8.8.8.8,9.9.9.9 # view multiple hosts

$ censys view platform.censys.io:80 # view a single web property
$ censys view platform.censys.io:80,google.com:80 # view multiple web properties

$ censys view 3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf # view a single certificate
$ censys view 3daf28...,123456... # view multiple certificates
```

## Asset Type Detection

The `view` command automatically detects the asset type based on the input format and fetches the corresponding data.

Defanged IPs and URLs are automatically refanged.

### Hosts
Hosts are identified by IPv4 or IPv6 addresses.

For example:
- `8.8.8.8`
- `8[.]8[.]8[.]8`
- `2001:4860:4860::8888`
- `2001[:]0db8[:]85a3[:]0000[:]0000[:]8a2e[:]0370[:]7334`


### Web Properties

Web properties are identified by `hostname:port` combinations, where `hostname` can be an IP address or a real hostname, like `platform.censys.io`. If the port is omitted, it will default to `443`.

For example:
- `platform.censys.io:80`
- `platform.censys.io` (port omitted, defaults to 443)
- `https://platform[.]censys[.]io:443`

### Certificates

Certificates are identified by their SHA-256 fingerprint (64-character hex string).

For example:

- `3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf`


## Flags

This section describes the flags available for the `view` command. To see global flags and how they might affect this command, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

### `--input-file`, `-i`

Read asset identifiers from a file instead of command-line arguments. Each line in the file should contain one asset identifier. If the file is `-`, read from standard input.

**Type:** `string`  
**Default:** none

```bash
$ censys view --input-file hosts.txt
$ cat hosts.txt | censys view --input-file -
```

### `--org-id`

Specify the organization ID to use for the request. This overrides the default organization ID from your configuration.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID (or the free-user wallet if not configured)

```bash
$ censys view --org-id 00000000-0000-0000-0000-000000000001 8.8.8.8
```

### `--at-time`, `--at`, `-a`

View data as of a specific point in time. The timestamp must be in RFC3339 format. Not supported for certificate assets.

**Type:** `string` (see [timestamps](#timestamps) for supported formats)
**Default:** none (current time)

```bash
$ censys view 8.8.8.8 --at-time 2025-09-15T14:30:00Z
```

## Output Formats

The `view` command defaults to **`json`** output format (or the global config value). You can override this with the `--output-format` flag (or `-O`).

**Default:** `json` (or configured global default)  
**Supported formats:** `json`, `yaml`, `tree`, `short`, `template`

### Format Descriptions

- **`json`** - Structured JSON output (default)
- **`yaml`** - Structured YAML output
- **`tree`** - Hierarchical tree view
- **`short`** - Concise summary view of assets
- **`template`** - Render using asset-specific Handlebars templates (see [Templates](#templates) section)

### Streaming Output

Use `--streaming` (or `-S`) to enable streaming mode, which outputs results as NDJSON (newline-delimited JSON) with one asset per line emitted immediately as data is fetched. This is useful when viewing many assets at once. See [global configuration](../GLOBAL_CONFIGURATION.md#--streaming--s) for more details.

### Examples

```bash
# Default: JSON output
$ censys view 8.8.8.8

# Short format: concise summary
$ censys view 8.8.8.8 --output-format short
$ censys view 8.8.8.8 -O short

# Template format: custom Handlebars rendering
$ censys view 8.8.8.8 --output-format template
$ censys view example.com:443 --output-format template

# YAML output
$ censys view 8.8.8.8 --output-format yaml
```

## Templates

`cencli` supports template-based rendering of raw data using the `--output-format template` flag, which allows you to define how you view your data. Templates are powered by **Handlebars v3** (via the [raymond](https://github.com/aymerick/raymond) Handlebars implementation).

![view-short](../../examples/view/view-short.gif)

When you first run `cencli`, default templates can be found in the configuration directory (typically `~/.config/cencli/templates/`) and are automatically created with sensible defaults on first use. Each asset type has its own template:

- **Host:** `host.hbs`
- **Certificate:** `certificate.hbs`
- **Web Property:** `webproperty.hbs`

To use templates, specify `--output-format template` (or `-O template`) when running the `view` command:

```bash
$ censys view 8.8.8.8 --output-format template
$ censys view example.com:443 -O template
```

### Customizing Templates

You can customize templates by editing the files in your templates directory. Alternatively, you can change the path to a template file in another location in your `config.yaml` file.

An easy way to create your own template is to take the raw data from a `view` command and place it into the [Handlebars Playground](https://handlebarsjs.com/playground.html). Make sure you are on version 3.0.

## Timestamps

The `--at-time` flag allows you to view historical data as it existed at a specific point in time. This feature is currently supported for **hosts** and **web properties** (not certificates).

### Supported Timestamp Formats

`cencli` accepts timestamps in multiple formats for flexibility:

**RFC3339 format (with timezone):**
```bash
$ censys view 8.8.8.8 --at-time "2025-09-15T14:30:00Z"
$ censys view 8.8.8.8 --at-time "2025-09-15T14:30:00-07:00"
```

**Date and time (uses default timezone):**
```bash
$ censys view 8.8.8.8 --at-time "2025-09-15 14:30:00"
```

**Date only (time defaults to 00:00:00 in default timezone):**
```bash
$ censys view 8.8.8.8 --at-time "2025-09-15"
$ censys view 8.8.8.8 --at-time "01/02/2006"
$ censys view 8.8.8.8 --at-time "2006/01/02"
```

**Date and time with explicit timezone offset:**
```bash
$ censys view 8.8.8.8 --at-time "2025-09-15 14:30:00 -0700"
$ censys view 8.8.8.8 --at-time "2025-09-15 14:30:00 -07:00"
```

### Default Timezone

When you provide a timestamp without timezone information (like `2025-09-15 14:30:00`), `cencli` interprets it using your configured default timezone.

The default timezone is `UTC` unless you configure a different one. See the [default timezone configuration](../GLOBAL_CONFIGURATION.md#default-timezone) for details on how to change this setting.

### Supported Timezones

`cencli` supports a curated list of common timezones. When configuring a default timezone, you must use one of the timezone strings from the supported list. See [timezones.go](../../internal/pkg/datetime/timezones.go) for the complete list of supported timezone identifiers.
