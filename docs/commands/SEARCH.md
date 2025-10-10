# Search Command

The `search` command allows you to perform search queries across hosts, certificates, and web properties from the Censys Platform.

![search](../../examples/search/search.gif)

## Usage

```bash
$ censys search "host.services: (protocol=SSH and not port: 22)" # complex query
$ censys search "services.port: 443" --fields host.ip,host.location # specify fields to return
$ censys search "services.service_name: HTTP" --max-pages -1 # fetch all pages
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
$ censys search "services.port: 443" --collection-id 550e8400-e29b-41d4-a716-446655440000
```

### `--org-id`

Specify the organization ID to use for the request. This overrides the default organization ID from your configuration.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID (or the free-user wallet if not configured)

```bash
$ censys search "services.port: 443" --org-id 00000000-0000-0000-0000-000000000001
```

### `--fields`, `-f`

Specify which fields to return in the response. This allows you to filter the output to only include the data you need, reducing response size and improving readability.

**Type:** `string` (comma-separated list)  
**Default:** none (returns all fields)

```bash
$ censys search "host.services: (protocol=SSH)" --fields host.ip,host.location
$ censys search "services.port: 443" -f host.ip,services.port,services.service_name
```

### `--page-size`, `-n`

The number of results to return per page. Larger page sizes reduce the number of API calls needed but may increase response time.

**Type:** `integer`  
**Default:** `100` (or [configured value](../GLOBAL_CONFIGURATION.md#searchpage-size))  
**Minimum:** `1`

```bash
$ censys search "services.port: 443" --page-size 50
```

### `--max-pages`, `-p`

The maximum number of pages to fetch. Use `-1` to fetch all available pages.

**Type:** `integer`  
**Default:** `1` (or [configured value](../GLOBAL_CONFIGURATION.md#searchmax-pages))  
**Special Values:** `-1` fetches all pages

```bash
$ censys search "services.port: 443" --max-pages 5
$ censys search "services.service_name: HTTP" --max-pages -1  # fetch all results
```

**Note:** Using `--max-pages -1` will fetch all available results, which may result in many API calls and take considerable time depending on the query.

### `--short`, `-s`

Render output using templates for a concise, human-readable summary instead of raw output. See the [using templates](#using-templates) section for more details.

**Type:** `boolean`  
**Default:** `false`

```bash
$ censys search "services.port: 443" --short
```

## Configuration

You can set default values for pagination flags in your [configuration file](../GLOBAL_CONFIGURATION.md#configuration-file):

```yaml
search:
  page_size: 100  # default number of results per page
  max_pages: 1    # default maximum number of pages to fetch
```

These defaults will be used when the corresponding flags are not specified on the command line.
