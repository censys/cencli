# DNS Command

The `dns` command allows you to look up Active DNS observations from the Censys Platform. For a domain name, it shows the records the name resolved to. For an IP address, it shows the domain names that resolved to it.

**Note:** This command is only available to organizations on the Censys Search and Censys Core plans.

![dns](../../examples/dns/dns.gif)

## Usage

```bash
$ censys dns censys.com # records active in the last 7 days
$ censys dns censys.com --record-type A,MX # only A and MX records
$ censys dns censys.com --timeline --duration 90d # each observed time range in the last 90 days
$ censys dns 104.18.10.84 # domain names that resolved to this IP
$ censys dns 104.18.10.84 --start 2026-06-01T00:00:00Z --duration 30d
$ censys dns 141.193.213.10 --timeline --domain censys.com # when did this domain point at this IP
$ censys dns censys.com,104.18.10.84 # several inputs, comma-separated
$ censys dns --input-file iocs.txt # read inputs from a file, one per line
$ censys dns --input-file - # read inputs from STDIN
$ censys dns censys.com --output-format json # raw data
$ censys dns censys.com -O template # custom Handlebars rendering
```

## Asset Type Detection

The `dns` command detects the lookup direction for each input:

- An IP address (IPv4 or IPv6, defanged or not) looks up the domain names that resolved to it.
- Anything else is read as a domain name. The command removes a scheme (`https://`, including defanged forms like `hxxps://` and `https[://]`), a path, and a trailing dot, and lowercases the name. Defanged names such as `censys[.]com` are accepted.
- A pasted URL whose host is an IP address (for example `https://8.8.8.8/`) is still looked up as an IP, not a domain name.

Several names or IPs can be given in one command, up to 100:

- As positional arguments. A comma-separated list within one argument (`censys.com,104.18.10.84`) is split into separate inputs. This splitting only applies to arguments — a comma inside a file line (see `--input-file` below) is kept as-is.
- An argument containing `//` (a URL, defanged or not, such as `https://censys.com/a,b` or `hxxp://censys.com`) is always read as one input, even if it contains a comma in its path or query.
- From a file, or from STDIN with `-`, using `--input-file`/`-i`. Each line is exactly one input. Lines are trimmed of surrounding whitespace, and blank lines are skipped.
- Duplicate inputs (after normalizing) are removed, keeping the first occurrence's position.
- More than 100 inputs after de-duplication is rejected with a `Too Many Assets` error.
- Every input is validated before any lookup runs. If one input is invalid, the whole command fails before any API call is made.

The following inputs are rejected:

- A port (`censys.com:443`) — a DNS name has no port; the error says to remove it.
- Brackets around an IPv6 address (`[2001:db8::1]`) — the error says to remove the brackets.
- A bare CIDR range (for example `8.8.8.8/32`, or defanged, `8[.]8[.]8[.]8/32` or `8.8.8.8[/]32`) — give one IP address instead. A URL with a numeric path (`https://8.8.8.8/32`, `hxxp://8.8.8.8/32`) is not a CIDR range; it is looked up as the IP `8.8.8.8`.
- An `@` in the name (for example `user@censys.com`).
- An empty label (for example `a..b.com`, or a trailing dot beyond the one FQDNs allow).

## Flags

This section describes the flags available for the `dns` command. To see global flags and how they might affect this command, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

### `--input-file`, `-i`

File to read the names or IPs from (or `-` for STDIN). **Overrides** positional arguments — if both are given, the file wins. Each line is one input: lines are trimmed of surrounding whitespace and blank lines are skipped, but a line is otherwise taken as-is. Unlike a positional argument, a file line cannot be shell-quoted, so it is never comma-split.

**Type:** `string` (path, or `-`)  
**Default:** none

```bash
$ censys dns --input-file iocs.txt
$ cat iocs.txt | censys dns --input-file -
```

### `--start`, `-s`

Start time for the DNS observation window in RFC3339 format.

**Type:** `string` (RFC3339 timestamp)  
**Default:** Calculated from `--end` and `--duration`, or current time minus duration if neither is specified

```bash
$ censys dns censys.com --start 2026-06-01T00:00:00Z --duration 30d
$ censys dns censys.com -s 2026-06-01T00:00:00Z --end 2026-06-30T00:00:00Z
```

**Note:** If both `--start` and `--end` are provided, they define the exact window (ignoring `--duration`).

### `--end`, `-e`

End time for the DNS observation window in RFC3339 format.

**Type:** `string` (RFC3339 timestamp)  
**Default:** Current time (or calculated from `--start` and `--duration`)

```bash
$ censys dns censys.com --end 2026-06-30T00:00:00Z --duration 30d
$ censys dns censys.com --start 2026-06-01T00:00:00Z -e 2026-06-30T00:00:00Z
```

**Note:** If both `--start` and `--end` are provided, they define the exact window (ignoring `--duration`).

### `--duration`, `-d`

Time window duration in human-readable format (e.g., `1d`, `1w`, `1y`, `2h`).

**Type:** `string` (human duration)  
**Default:** `7d` (7 days)

```bash
$ censys dns censys.com --duration 30d
$ censys dns censys.com -d 1w
$ censys dns 104.18.10.84 --duration 90d
```

**Supported units:**
- `h` - hours
- `d` - days
- `w` - weeks (7 days)
- `y` - years (365 days)

**How duration works:**
- If only `--duration` is specified: window is from (now - duration) to now
- If `--start` is specified: window is from start to (start + duration)
- If `--end` is specified: window is from (end - duration) to end
- If both `--start` and `--end` are specified: duration is ignored

**Note:** The duration must be greater than 0; a negative or zero duration is rejected.

### `--timeline`, `-t`

Show one row for each observed time range of a record, instead of one row for each record.

**Type:** `boolean`  
**Default:** `false`

```bash
$ censys dns censys.com --timeline --duration 90d
```

### `--domain`

Limit an IP timeline to one domain name that resolved to it — "when did this domain point at this IP". It works only for IP input with `--timeline`; any other use (a domain name among the inputs, or `--timeline` not set) is a usage error. An explicitly empty value (`--domain ""`) is also a usage error, rather than being treated as unset.

**Type:** `string` (domain name)  
**Default:** none

```bash
$ censys dns 141.193.213.10 --timeline --domain censys.com
```

`censys dns <name> --timeline -r A` answers the same question from the name side: when an A record for that name pointed at a given IP.

### `--record-type`, `-r`

The record types to include. Domain names support `A`, `AAAA`, `MX`, `NS`, `SOA`, and `TXT`. IP addresses support `A` and `AAAA`. Values are not case-sensitive.

**Type:** `string` (comma-separated list)  
**Default:** none (returns all supported types)

```bash
$ censys dns censys.com --record-type A,MX
$ censys dns 104.18.10.84 -r AAAA
```

### `--page-size`, `-n`

The number of records to return per page. Larger page sizes reduce the number of API calls needed but may increase response time.

**Type:** `integer`  
**Default:** `100`  
**Minimum:** `1`  
**Maximum:** `100`

```bash
$ censys dns censys.com --page-size 50
```

### `--max-pages`, `-p`

The maximum number of pages to fetch. Use `-1` to fetch all available pages.

**Type:** `integer`  
**Default:** `10`  
**Special Values:** `-1` fetches all pages

```bash
$ censys dns 104.18.10.84 --max-pages 20
$ censys dns 104.18.10.84 --max-pages -1  # fetch all results
```

**Note:** Using `--max-pages -1` will fetch all available results, which may result in many API calls and take considerable time for an IP address that many domain names resolved to. With several inputs, this applies to each input separately.

### `--org-id`

Specify the organization ID to use for the request. This overrides the default organization ID from your configuration.

> [!IMPORTANT]
> This flag applies **only to personal access tokens**, which are not organization-scoped. If you authenticated with `censys auth login`, the organization is fixed by that login and passing `--org-id` **fails with an error** — see [Organization context](AUTH.md#organization-context).

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID (or the free-user wallet if not configured). With an OAuth login, the organization that login was authorized for.

```bash
$ censys dns censys.com --org-id 00000000-0000-0000-0000-000000000001
```

## Output Formats

The `dns` command defaults to **`short`** output format, which displays results as a formatted table. You can override this with the `--output-format` flag (or `-O`).

**Default:** `short` (table view)  
**Supported formats:** `json`, `yaml`, `tree`, `short`, `template`

### Template output

Use `--output-format template` (or `-O template`) to render results with a custom Handlebars template. The `dns` command's template entity is `dns`, and its default template file is `dns.hbs`. When cencli loads its configuration and `templates.dns.path` is not set, it looks in your templates directory (`~/.config/cencli/templates/`, or `$CENCLI_DATA_DIR/templates/`) for a file named `dns.*`. If there is none, it writes the default `dns.hbs` there. An existing file is never overwritten, so a template written by an earlier version keeps its content; delete it to get the current default.

The template receives the same flat list of records as JSON output, with the same field names. Each record also has every field the other record types have (empty when this record lacks it), so a field missing on one record never picks up another record's value, and a `timeline` field that is `true` for `--timeline` records. Every string value has its control characters removed before rendering; the default template then prints values as-is (with `{{{ }}}`, not HTML-escaped), so quotes and ampersands in TXT, SPF, and DKIM values appear unchanged.

To use your own template, point `templates.dns.path` at it in `config.yaml`:

```yaml
templates:
  dns:
    path: /path/to/custom/dns.hbs
```

```bash
$ censys dns censys.com --output-format template
$ censys dns censys.com -O template
```

See [global configuration](../GLOBAL_CONFIGURATION.md#templates) for more on customizing templates.

### JSON, YAML, and NDJSON output

Every record in structured output (`json`, `yaml`, `--streaming` NDJSON) has an `input` field naming the normalized name or IP it answers. With several inputs, data output is one flat array covering every input, not grouped per input.

### Streaming Output

Use `--streaming` (or `-S`) to enable streaming mode, which outputs results as NDJSON (newline-delimited JSON) with one record per line emitted immediately as data is fetched. This is useful with `--max-pages -1`. See [global configuration](../GLOBAL_CONFIGURATION.md#--streaming--s) for more details.

### Examples

```bash
# Default: formatted table view
$ censys dns censys.com

# JSON output
$ censys dns censys.com --output-format json

# NDJSON output (one record per line)
$ censys dns censys.com --streaming

# Template output
$ censys dns censys.com -O template
```

## Understanding the Output

- **The time window selects records; it does not trim their dates.** A record appears if it was active in the window. `First Seen` and `Last Seen` cover the full observed life of the record, so `First Seen` can be earlier than the window start.
- **The window line** under the title shows the window in UTC. The default window is the last 7 days, so older records do not appear unless you widen it with `--duration`. With several inputs, the window line is printed once, followed by one section (title and table) per input, in input order.
- **Timeline mode** (`--timeline`) shows `First Observed` and `Last Observed` for each separate time range in which a record was seen.
- **Values:** MX shows `priority server`. SOA shows `mname rname`. The table shortens long TXT values; JSON output keeps the full value.
- **A count like `(1000 of 5321)`** means not all matching records were fetched. The note printed under the table (suppressed by `--quiet`) says when `--max-pages` is the reason (use `--max-pages -1` to fetch all records); a fetch that failed partway through instead prints the error that stopped it.
- **With several inputs, a failing input does not stop the others.** Errors are printed to stderr after the output: first the errors of fetches that failed partway through (partial pages), then the failed inputs, each group in input order. The command exits `0` if at least one input succeeded, except after a plan error (403), which prints what was already fetched and then exits non-zero. If every input fails, every error is printed in input order and the command exits non-zero.
- **A plan error (403) stops further lookups**, since a plan restriction applies to every input the same way — but it keeps and prints whatever was already fetched for earlier inputs, and the errors of earlier failed inputs, before reporting the 403 last.
- **An interrupt** (Ctrl-C) reports how many inputs were not looked up (for example, `interrupted; 3 inputs not looked up`).
- **With several inputs, response metadata is combined into one line** (instead of one block per input) reporting the last request's method, URL, and status, plus the latency and page count summed across every lookup.

### `dns` and `view`

`censys view <ip>` shows the forward and reverse DNS names in a host's current record. `censys dns` shows Active DNS observations over a time window. The two data sources can show different names. A short-format `view` of a single host prints a tip to stderr pointing to `censys dns <ip>` (suppressed by `--quiet`).
