# DNS Command

The `dns` command allows you to look up Active DNS observations from the Censys Platform. For a domain name, it shows the records the name resolved to. For an IP address, it shows the domain names that resolved to it.

**Note:** This command is only available to organizations on the Censys Search and Censys Core plans.

## Usage

```bash
$ censys dns censys.com # records active in the last 7 days
$ censys dns censys.com --record-type A,MX # only A and MX records
$ censys dns censys.com --timeline --duration 90d # each observed time range in the last 90 days
$ censys dns 104.18.10.84 # domain names that resolved to this IP
$ censys dns censys.com --output-format json # raw data
```

## Asset Type Detection

The `dns` command detects the lookup direction from the input:

- An IP address (IPv4 or IPv6, defanged or not) looks up the domain names that resolved to it.
- Anything else is read as a domain name. The command removes a scheme (`https://`), a path, and a trailing dot, and changes the name to lowercase. Defanged names such as `censys[.]com` are accepted.
- A port (`censys.com:443`) is rejected: DNS names have no port. The command takes one input only.

## Flags

This section describes the flags available for the `dns` command. To see global flags and how they might affect this command, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

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

### `--timeline`, `-t`

Show one row for each observed time range of a record, instead of one row for each record.

**Type:** `boolean`  
**Default:** `false`

```bash
$ censys dns censys.com --timeline --duration 90d
```

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

**Note:** Using `--max-pages -1` will fetch all available results, which may result in many API calls and take considerable time for an IP address that many domain names resolve to.

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
**Supported formats:** `json`, `yaml`, `tree`, `short`

**Note:** The `template` output format is **not supported** for the dns command.

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
```

## Understanding the Output

- **The time window selects records; it does not trim their dates.** A record appears if it was active in the window. `First Seen` and `Last Seen` cover the full observed life of the record, so `First Seen` can be earlier than the window start.
- **The window line** under the title shows the window in UTC. The default window is the last 7 days, so older records do not appear unless you widen it with `--duration`.
- **Timeline mode** (`--timeline`) shows `First Observed` and `Last Observed` for each separate time range in which a record was seen.
- **Values:** MX shows `priority server`. SOA shows `mname rname`. The table shortens long TXT values; JSON output keeps the full value.
- **A count like `(1000 of 5321)`** means not all matching records were fetched. The note printed under the table says when `--max-pages` is the reason (use `--max-pages -1` to fetch all records); a fetch that failed partway through instead prints the error that stopped it.

### `dns` and `view`

`censys view <ip>` shows the forward and reverse DNS names in a host's current record. `censys dns` shows Active DNS observations over a time window. The two data sources can show different names.
