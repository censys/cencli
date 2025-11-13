# History Command (WIP)

The `history` command allows you to retrieve historical data for hosts, web properties, and certificates from the Censys Platform. This command provides time-series data showing how assets have changed over time.

![history](../../examples/history/history.gif)

## Usage

```bash
$ censys history 8.8.8.8 --duration 30d # host history for last 30 days
$ censys history example.com:443 --start 2025-01-01T00:00:00Z --duration 7d # web property history
$ censys history 3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf --end 2025-05-31T00:00:00Z --duration 72d # certificate history
```

## Asset Type Detection

The `history` command automatically detects the asset type based on the input format and fetches the corresponding historical data. See the [view command](VIEW.md#asset-type-detection) for more information on asset type detection.

## Flags

This section describes the flags available for the `history` command. To see global flags and how they might affect this command, see the [global configuration docs](mdc:../GLOBAL_CONFIGURATION.md).

### `--start`, `-s`

Start time for the historical data window in RFC3339 format.

**Type:** `string` (RFC3339 timestamp)  
**Default:** Calculated from `--end` and `--duration`, or current time minus duration if neither is specified

```bash
$ censys history 8.8.8.8 --start 2025-01-01T00:00:00Z --duration 30d
$ censys history example.com:443 -s 2025-01-01T00:00:00Z --end 2025-01-31T00:00:00Z
```

**Note:** If both `--start` and `--end` are provided, they define the exact window (ignoring `--duration`).

### `--end`, `-e`

End time for the historical data window in RFC3339 format.

**Type:** `string` (RFC3339 timestamp)  
**Default:** Current time (or calculated from `--start` and `--duration`)

```bash
$ censys history 8.8.8.8 --end 2025-05-31T00:00:00Z --duration 30d
$ censys history example.com:443 --start 2025-01-01T00:00:00Z -e 2025-01-31T00:00:00Z
```

**Note:** If both `--start` and `--end` are provided, they define the exact window (ignoring `--duration`).

### `--duration`, `-d`

Time window duration in human-readable format (e.g., `1d`, `1w`, `1y`, `2h`).

**Type:** `string` (human duration)  
**Default:** `7d` (7 days)

```bash
$ censys history 8.8.8.8 --duration 30d
$ censys history example.com:443 -d 1w
$ censys history 3daf28... --duration 90d
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

### `--org-id`

Specify the organization ID to use for the request. This overrides the default organization ID from your configuration.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID (or the free-user wallet if not configured)

```bash
$ censys history 8.8.8.8 --org-id 00000000-0000-0000-0000-000000000001
```

## Output Formats

The `history` command defaults to **`json`** output format (or the global config value). Unlike other commands, history only supports structured data formats.

**Default:** `json` (or configured global default)  
**Supported formats:** `json`, `yaml`, `ndjson`, `tree`

**Note:** The `short` and `template` output formats are **not supported** for the history command due to the time-series nature of the data.

### Examples

```bash
# Default: JSON output
$ censys history 8.8.8.8 --duration 30d

# YAML output
$ censys history 8.8.8.8 --duration 30d --output-format yaml

# NDJSON output (one event per line)
$ censys history 8.8.8.8 --duration 30d --output-format ndjson
```

## Output Format

The `history` command outputs raw JSON arrays containing the historical information. The structure varies by asset type:

### Host History Output

Returns an array of timeline events:

```json
[
  {
    "event_time": "2025-01-15T12:34:56Z",
    "event_type": "host_observed",
    "services": [...],
    ...
  },
  {
    "event_time": "2025-01-16T08:22:10Z",
    "event_type": "host_observed",
    ...
  }
]
```

### Certificate History Output

Returns an array of observation ranges showing when and where the certificate was seen:

```json
[
  {
    "ip": "1.2.3.4",
    "port": 443,
    "transport_protocol": "tcp",
    "protocols": ["https"],
    "start_time": "2025-01-01T00:00:00Z",
    "end_time": "2025-01-15T23:59:59Z",
    ...
  },
  {
    "ip": "5.6.7.8",
    "port": 443,
    ...
  }
]
```

### Web Property History Output

Returns an array of daily snapshots:

```json
[
  {
    "Time": "2025-01-01T00:00:00Z",
    "Exists": true,
    "Data": {
      "hostname": "example.com",
      "port": 443,
      "endpoints": [...],
      "cert": {...},
      ...
    }
  },
  {
    "Time": "2025-01-02T00:00:00Z",
    "Exists": true,
    "Data": {...}
  },
  {
    "Time": "2025-01-03T00:00:00Z",
    "Exists": false,
    "Data": null
  }
]
```

**Note:** Web property snapshots include an `Exists` field indicating whether the property had meaningful data at that time. If `Exists` is `false`, the `Data` field will be `null`.

## Performance Notes

Historical data fetching can be time-intensive, especially for:
- **Web properties** with long time windows (fetches daily snapshots)
- **Hosts** with many timeline events (requires pagination)
- **Certificates** with many observations across hosts

The command has **no timeout** by default to accommodate long-running requests.
