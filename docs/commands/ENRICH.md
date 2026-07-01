# Enrich Command

The `enrich` command looks up host IPs through the Censys **Host Enrichment API** — a new, lightweight API purpose-built to help security teams operationalize external internet data without the friction of traditional credit consumption. It provides a curated, fixed subset of host IPv4/IPv6 data designed specifically for high-volume, automated lookups in SOC environments, such as SIEM and SOAR integrations.

This feature is available to customers on the **Censys Core** plan.

![enrich](../../examples/enrich/enrich-short.gif)

## Usage

```bash
$ censys enrich 104.168.107.43 # enrich a single host
$ censys enrich 104.168.107.43,8.8.8.8 # enrich multiple hosts
$ censys enrich --input-file ips.txt # enrich every IP in a file
$ cat ips.txt | censys enrich --input-file - # read IPs from STDIN
```

The command accepts one or more host IPs as a comma-separated argument or via `--input-file`. Each IP is enriched independently and the results are returned together. Defanged IPs (e.g. `104[.]168[.]107[.]43`) are automatically refanged.

> **Note:** enrichment is host-only. To look up certificates or web properties, or to retrieve the full host asset, use the [`view` command](./VIEW.md).

## What you get

Unlike `view`, which returns the full host asset, `enrich` returns a curated, fixed subset of host data optimized for triage and automation:

- Location and autonomous system
- WHOIS
- DNS (forward and reverse)
- Labels
- Reputation
- Network and privacy classifications
- A trimmed service list (port / protocol / labels / threats)
- Third-party verdicts (MalloryAI)

Each field is included only when Censys has data for it, so the set returned for a given host may be sparse — or effectively empty for hosts with no enrichment data available.

## Requirements

- **Censys Core plan** with the Host Enrichment entitlement enabled.
- An **organization ID** is required. Provide it with `--org-id` or configure a default (see the [config command docs](./CONFIG.md)). If no organization is available, the command fails before making any request.
- **No credits are consumed** by enrichment lookups. The endpoint enforces a daily request limit; once it is reached the API returns a rate-limit response and the command reports a clear "daily enrichment limit reached" message.

## Flags

This section describes the flags available for the `enrich` command. To see global flags and how they might affect this command, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

### `--input-file`, `-i`

Read host IPs from a file instead of command-line arguments. Each line in the file should contain one IP. If the file is `-`, read from standard input. Overrides the positional argument.

**Type:** `string`  
**Default:** none

```bash
$ censys enrich --input-file ips.txt
$ cat ips.txt | censys enrich --input-file -
```

### `--org-id`

Specify the organization ID to use for the request. This overrides the default organization ID from your configuration. Enrichment requires an organization, so this flag (or a configured default) is mandatory.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID

```bash
$ censys enrich --org-id 00000000-0000-0000-0000-000000000001 104.168.107.43
```

## Multiple IPs

Because the endpoint is single-IP, the command fans out concurrent lookups when given several IPs — a fit for the high-volume SOC use case. Per-IP failures do not abort the run: successfully enriched hosts are still returned, and a summary of how many failed is reported to stderr.

## Output Formats

The `enrich` command defaults to **`json`** output format (or the global config value). You can override this with the `--output-format` flag (or `-O`).

**Default:** `json` (or configured global default)  
**Supported formats:** `json`, `yaml`, `tree`, `short`

### Format Descriptions

- **`json`** - Structured JSON output (default)
- **`yaml`** - Structured YAML output
- **`tree`** - Hierarchical tree view
- **`short`** - Concise, SOC-friendly summary of each host

![enrich-short](../../examples/enrich/enrich-short.gif)

### Streaming Output

Use `--streaming` (or `-S`) to enable streaming mode, which outputs results as NDJSON (newline-delimited JSON) with one host per line, emitted immediately as each lookup completes. This is useful when enriching many IPs at once. Under streaming, results arrive in completion order rather than input order. See [global configuration](../GLOBAL_CONFIGURATION.md#--streaming--s) for more details.

### Examples

```bash
# Default: JSON output
$ censys enrich 104.168.107.43

# Short format: concise SOC summary
$ censys enrich 104.168.107.43 --output-format short
$ censys enrich 104.168.107.43 -O short

# YAML output
$ censys enrich 104.168.107.43 --output-format yaml

# High-volume: stream NDJSON as each lookup completes
$ censys enrich --input-file ips.txt --streaming
```
