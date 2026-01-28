# Search Command

The `search` command allows you to perform search queries across hosts, certificates, and web properties from the Censys Platform.

![search](../../examples/search/search.gif)

## Usage

```bash
$ censys search "host.services: (protocol=SSH and not port: 22)" # complex query
$ censys search "host.services.port: 443" --fields host.ip,host.location # specify fields to return
$ censys search "host.services.protocol: 'HTTP'" --max-pages -1 # fetch all pages
```

## Query Syntax

The search command uses Censys Query Language (CQL) to filter and find assets. Queries follow a field-based syntax with support for logical operators, wildcards, and more.

For detailed information about query syntax and available fields, see the [Censys Query Language documentation](https://docs.censys.com/docs/censys-query-language).


## Flags

This section describes the flags available for the `search` command. To see global flags and how they might affect this command, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

### `--collection-id`, `-c`

Search within a specific collection instead of globally.

**Type:** `string` (UUID format)  
**Default:** none (searches globally)

```bash
$ censys search "host.services.port: 443" --collection-id 550e8400-e29b-41d4-a716-446655440000
```

### `--org-id`

Specify the organization ID to use for the request. This overrides the default organization ID from your configuration.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID (or the free-user wallet if not configured)

```bash
$ censys search "host.services.port: 443" --org-id 00000000-0000-0000-0000-000000000001
```

### `--fields`, `-f`

Specify which fields to return in the response. This allows you to filter the output to only include the data you need, reducing response size and improving readability.

**Type:** `string` (comma-separated list)  
**Default:** none (returns all fields)

```bash
$ censys search "host.services.protocol=SSH" --fields host.ip,host.location.country
$ censys search "host.services.port: 443" -f host.services.port,host.ip,host.services.protocol
```

### `--page-size`, `-n`

The number of results to return per page. Larger page sizes reduce the number of API calls needed but may increase response time.

**Type:** `integer`  
**Default:** `100` (or [configured value](../GLOBAL_CONFIGURATION.md#searchpage-size))  
**Minimum:** `1`

```bash
$ censys search "host.services.port: 443" --page-size 50
```

### `--max-pages`, `-p`

The maximum number of pages to fetch. Use `-1` to fetch all available pages.

**Type:** `integer`  
**Default:** `1` (or [configured value](../GLOBAL_CONFIGURATION.md#searchmax-pages))  
**Special Values:** `-1` fetches all pages

```bash
$ censys search "host.services.port: 443" --max-pages 5
$ censys search "host.services.protocol: HTTP" --max-pages -1  # fetch all results
```

**Note:** Using `--max-pages -1` will fetch all available results, which may result in many API calls and take considerable time depending on the query.

## Output Formats

The `search` command defaults to **`json`** output format (or the global config value). You can override this with the `--output-format` flag (or `-O`).

**Default:** `json` (or configured global default)  
**Supported formats:** `json`, `yaml`, `tree`, `short`, `template`

### Format Descriptions

- **`json`** - Structured JSON output (default)
- **`yaml`** - Structured YAML output
- **`tree`** - Hierarchical tree view
- **`short`** - Concise summary view of search results
- **`template`** - Render using Handlebars templates

### Examples

```bash
# Default: JSON output
$ censys search "host.services.port: 443"

# Short format: concise summary
$ censys search "host.services.port: 443" --output-format short
$ censys search "host.services.port: 443" -O short

# Template format: custom Handlebars rendering
$ censys search "host.services.port: 443" --output-format template

# YAML output
$ censys search "host.services.port: 443" --output-format yaml
```

For more information on customizing templates, see the [view command templates documentation](VIEW.md#templates).

## Streaming Output

When using `--streaming` (or `-S`), results are **streamed immediately** as NDJSON (newline-delimited JSON) as they are fetched from the API. This provides several benefits for large result sets:

- **Output begins before all pages are fetched** - You see results as soon as the first page is retrieved
- **Memory usage stays bounded** - Results are written immediately rather than accumulated in memory
- **Partial results are preserved on interruption** - If you press Ctrl-C or an error occurs, all previously emitted records remain intact
- **Safe for large queries** - Ideal for use with `--max-pages -1` when fetching potentially unbounded result sets

### Streaming Example

```bash
# Stream all SSH hosts to a file
$ censys search "host.services.port: 22" --max-pages -1 --streaming > ssh_hosts.jsonl

# Process results as they arrive using jq
$ censys search "host.services.port: 443" --max-pages 10 -S | jq -r '.host.ip'

# Count results without storing them all in memory
$ censys search "host.services.protocol: HTTP" --max-pages -1 -S | wc -l
```

### When to Use Streaming

Use `--streaming` when:
- Fetching many pages of results (`--max-pages -1` or large values)
- Piping output to other tools that process line-by-line
- Writing large result sets to files
- You want to see results as they arrive rather than waiting for completion

Use other formats (`json`, `yaml`, `tree`) when:
- Working with small, bounded result sets
- You need the complete data structure (e.g., for JSON array processing)
- Displaying results in a terminal for human reading

**Note:** `--streaming` cannot be used together with `--output-format`. You can also enable streaming globally by setting `streaming: true` in your config file.

## Configuration

You can set default values for pagination flags in your [configuration file](../GLOBAL_CONFIGURATION.md#configuration-file):

```yaml
search:
  page_size: 100  # default number of results per page
  max_pages: 1    # default maximum number of pages to fetch
```

These defaults will be used when the corresponding flags are not specified on the command line.
