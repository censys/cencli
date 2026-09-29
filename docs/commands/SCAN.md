# Scan Command

The `scan` command requests a live rescan of a web property and tracks scans until they complete. A rescan refreshes what Censys knows about the web property; once it completes, `censys view <hostname:port>` shows the fresh data.

Scanning requires an organization (Enterprise). **A rescan costs 10 credits per accepted request**; reading a scan's status is free.

Running `censys scan` without a subcommand prints help.

## Usage

```bash
$ censys scan rescan example.com:443                 # request a rescan (10 credits), print the scan ID
$ censys scan rescan example.com:443 --wait          # request a rescan and poll until it completes
$ censys scan get <scan-id>                          # show a scan's current status (free)
$ censys scan get <scan-id> --wait                   # poll an existing scan until it completes
```

## Organization Context

Every `scan` subcommand accepts **`--org-id`, `-o`** (type `string`, UUID format), and every one **requires** an organization. Unlike lookups, scans have no free-account fallback.

- **Personal access token** — the stored organization ID by default, or `--org-id` per subcommand. To store a default, run `censys config org-id add` (see the [config command docs](./CONFIG.md)). With neither, the command fails before sending anything (exit 2).
- **OAuth login (`censys auth login`)** — the organization is fixed by what that login was authorized for, and `--org-id` fails with an error. A login authorized for your free account cannot scan (exit 1). See [Organization context](AUTH.md#organization-context).

To see global flags and how they affect these commands, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

## Commands

### `scan rescan`

Request one live rescan of a web property.

```bash
$ censys scan rescan example.com:443
$ censys scan rescan https://example.com              # scheme stripped; port defaults to 443
$ censys scan rescan example.com:8443 --wait --timeout 5m
```

The target is a web property, `hostname:port`, detected the same way [`view`](VIEW.md#asset-type-detection) does it. Hosts (a bare IP) and certificates are rejected with a usage error (exit 2) before anything is sent: a host rescan needs the service's protocol, which the CLI cannot know without a lookup.

Without `--wait` the command prints the new scan and how to track it, then exits:

```console
Track with: censys scan get 3f2b9c1e-… --wait
```

With `--wait` it polls until the scan completes and prints only the final state.

#### Credits and retries

Each accepted rescan costs 10 credits. The request is **sent exactly once and never retried automatically**, even though other commands retry server errors: a server error or dropped connection can arrive after the rescan was already accepted, and retrying could charge you twice. In that case the command tells you the credits **may** have been charged and to check `censys org credits` before trying again. The same applies if you interrupt the command while the request is in flight — there is no scan ID yet to track.

#### Flags

**`--wait`, `-w`**: Poll until the scan completes.

**Type:** `boolean`  
**Default:** `false`

**`--timeout`**: How long to wait before giving up. Requires `--wait`. Use `0` for no limit.

**Type:** `string` (duration, e.g. `5m`, `1h`)  
**Default:** `15m`

### `scan get`

Show the current state of a tracked scan by its UUID. This works for any tracked scan — rescans and discovery scans, web and host targets, including ones started outside the CLI — and prints whichever target the scan has.

```bash
$ censys scan get <scan-id>
$ censys scan get <scan-id> --wait
$ censys scan get <scan-id> --output-format json
```

Without `--wait` this is a plain read and **exits 0**, even for a scan whose tasks were rejected. A scan ID that is not a UUID is a usage error (exit 2).

#### Flags

Same `--wait` and `--timeout` flags as `scan rescan`.

## Waiting

A rescan is accepted in seconds and usually completes in a few minutes (about two and a half for `platform.censys.io:80`). `--wait` polls the scan behind a spinner, starting at 5 seconds between polls and backing off to 30.

A completed scan **succeeded** if at least one of its tasks ended `scanned` or `completed`. If every task ended `rejected`, `timed_out`, or `ignored` — or the scan ran no tasks — the scan is still printed, then the command exits 1.

If `--timeout` expires or you press Ctrl-C, the command stops polling, prints the last state it saw, and tells you how to pick tracking back up with `censys scan get <scan-id> --wait`. `scan rescan` falls back to the accepted request if no poll finished. `scan get` prints nothing if the wait ended before its first poll returned. The scan itself keeps running server-side.

## Output Formats

`scan` commands default to **`short`** output. Override with `--output-format` (or `-O`).

**Default:** `short`  
**Supported formats:** `short`, `json`, `yaml`, `tree`

The short view is a detail block followed by the scan's tasks, as the API reports them. For example, a completed rescan of `platform.censys.io:80`:

```console
$ censys scan get 9f2daf0c-10bf-4909-838f-bdf89a354e57

━━━ Tracked Scan ━━━

  ID:           9f2daf0c-10bf-4909-838f-bdf89a354e57
  Target:       platform.censys.io:80 (web_origin)
  Created:      2026-09-29T16:21:01Z
  Completed:    yes

Task                   Status      Updated

Rescanning HTTP at / | completed | 2026-09-29T16:23:36Z
```

In `json`/`yaml` the scan is an object with `tracked_scan_id`, `completed`, `create_time`, `target` (exactly as the API returns it: `web_origin`, `service_id`, `host_port`, or `hostname_port`), and `tasks[]` of `{description, status, update_time}`. With `--wait`, stdout holds a single scan object — the final one, or the last one seen if the wait ended early.

Templates (`-O template`) are not supported for `scan`.

## Exit Codes

| Code | Meaning |
| ---- | ------- |
| 0    | Success. Without `--wait`, any successful read — including a rejected scan |
| 1    | API error, missing credentials, a free-account login, an uncertain rescan failure, or a waited-on scan that completed without results |
| 2    | Usage error before anything is sent — a host or certificate given to `scan rescan`, a scan ID that is not a UUID, `--timeout` without `--wait`, a negative `--timeout`, `--org-id` with an OAuth login, or no organization configured |
| 124  | `--wait` gave up after `--timeout` (the last state seen is still printed if a poll had succeeded, and `scan rescan` always prints the accepted request) |
| 130  | Interrupted. During a wait, the output is the same as for 124. If the rescan request itself is interrupted, nothing is printed and you are told the credits may have been charged |

## Limitations

- **Web properties only.** Rescanning a host service is not supported yet.
- Hostname validation is local only as far as the asset parser goes (scheme, port, IPv6 brackets, a dot in the hostname). Internationalized names and length limits are checked by the API, which rejects them with an error naming the problem.
