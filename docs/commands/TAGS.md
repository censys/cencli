# Tags Command

The `tags` command manages tags and tag assignments for your organization. Tags let you label and organize assets (hosts, certificates, and web properties) for tracking and filtering.

Running `censys tags` without a subcommand prints help.

## Usage

```bash
$ censys tags list                            # list all tags
$ censys tags get my-tag                      # show one tag
$ censys tags create my-tag                   # create a tag
$ censys tags update my-tag --privacy shared  # change a tag
$ censys tags delete my-tag                   # delete a tag

$ censys tags assign my-tag 8.8.8.8           # tag an asset
$ censys tags unassign my-tag 8.8.8.8         # untag an asset
$ censys tags assignments my-tag              # list what a tag is assigned to

$ censys tags operations list                 # list bulk jobs
$ censys tags operations get my-tag <op-id>   # inspect one bulk job
$ censys tags operations cancel my-tag <op-id> # stop a running bulk job
```

## Organization Context

Every `tags` subcommand accepts **`--org-id`, `-o`** (type `string`, UUID format). Which organization the commands act on depends on how you authenticated:

- **Personal access token** — a PAT is not organization-scoped, so you choose: the stored organization ID by default, or `--org-id` per subcommand. To store a default, run `censys config org-id add` (see the [config command docs](./CONFIG.md)).
- **OAuth login (`censys auth login`)** — the organization is fixed by what that login was authorized for. Stored organization IDs are ignored and `--org-id` fails with an error; run `censys auth logout` and log in again to target a different organization. See [Organization context](AUTH.md#organization-context).

To see global flags and how they affect these commands, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

## Tag and Asset Identifiers

**Tags** are identified by **name or UUID**, interchangeably — tag names are unique within an organization. `tags get` hands the identifier straight to the API, which resolves names itself. Every other command needs the tag's UUID, so passing a name costs one extra lookup to resolve it; passing a UUID never does.

**Assets** are identified by type:

| Asset type   | Identifier                                | Example           |
| ------------ | ----------------------------------------- | ----------------- |
| Host         | IPv4 or IPv6 address                      | `8.8.8.8`         |
| Web property | `hostname:port`                           | `example.com:443` |
| Certificate  | SHA-256 fingerprint (64 hex characters)   | —                 |

The type is detected from the format, the same way [`view`](VIEW.md#asset-type-detection) does it. Asset types **can be mixed** in a single `assign` or `unassign` call.

## Commands

### `tags list`

List all tags in your organization.

```bash
$ censys tags list                        # list all tags
$ censys tags list --privacy shared       # only shared tags
$ censys tags list --name my-tag          # filter by exact name
$ censys tags list --order-by name_desc   # sort by name, descending
$ censys tags list --output-format json   # output as JSON
```

Results are paginated. By default only the first page is fetched; use `--max-pages` to fetch more, or `-1` for all pages. When more tags exist than were fetched, the `short` output header reads `Tags (N of TOTAL)`.

#### Flags

**`--privacy`**: Filter by privacy setting. `private` tags are visible to and editable by organization admins only; `shared` tags are visible to all organization members.

**Type:** `string` (`private` | `shared`)  
**Default:** none (no privacy filter)

**`--name`**: Filter by exact tag name.

**Type:** `string`  
**Default:** none

**`--created-by`**: Filter by the UUID of the tag's creator.

**Type:** `string` (UUID format)  
**Default:** none

**`--order-by`**: Sort order for the results.

**Type:** `string` (`name_asc`, `name_desc`, `created_at_asc`, `created_at_desc`, `updated_at_asc`, `updated_at_desc`)  
**Default:** none (the API's own ordering)

**`--page-size`, `-n`**: Number of tags to return per page. The API caps this at 1000.

**Type:** `integer` (1–1000)  
**Default:** `100`

**`--max-pages`, `-p`**: Maximum number of pages to fetch. Use `-1` to fetch all pages.

**Type:** `integer`  
**Default:** `1`

### `tags get`

Retrieve a single tag by its name or UUID.

```bash
$ censys tags get my-tag                       # get a tag by name
$ censys tags get <tag-id>                     # get a tag by UUID
$ censys tags get my-tag --asset-count         # also report how many assets are tagged
$ censys tags get my-tag --output-format json  # output as JSON
```

#### Flags

**`--asset-count`**: Also report how many assets the tag is assigned to.

The tag record itself carries no assignment count, so this costs an **additional request**, which is why it is opt-in rather than automatic. If that second request fails, the tag is still printed and the count error is reported afterwards — a failed count never fails the command.

**Type:** `boolean`  
**Default:** `false`

### `tags create`

Create a new tag with the given name. Tag names must be unique within an organization.

```bash
$ censys tags create my-tag                                              # create a private tag
$ censys tags create my-tag --privacy shared                             # create a shared tag
$ censys tags create my-tag --description "Assets flagged for review"    # create a tag with a description
```

#### Flags

**`--privacy`**: Tag visibility. New tags are private by default; use `shared` to make a tag visible to all organization members.

**Type:** `string` (`private` | `shared`)  
**Default:** `private`

**`--description`**: A human-readable description of the tag.

**Type:** `string`  
**Default:** none

### `tags update`

Update an existing tag by its name or UUID. **At least one mutation flag is required** — an update with nothing to change is rejected rather than sent.

```bash
$ censys tags update my-tag --description "Assets flagged for review"  # set a description
$ censys tags update my-tag --privacy shared                           # make a tag visible to the organization
$ censys tags update my-tag --name renamed-tag                         # rename a tag
$ censys tags update my-tag --clear-description                        # remove the description
```

#### Flags

**`--name`**: A new name for the tag.

**Type:** `string`  
**Default:** none (name unchanged)

**`--privacy`**: New tag visibility.

**Type:** `string` (`private` | `shared`)  
**Default:** none (privacy unchanged)

**`--description`**: A new description for the tag. Cannot be combined with `--clear-description`.

**Type:** `string`  
**Default:** none (description unchanged)

**`--clear-description`**: Remove the tag's description. Cannot be combined with `--description`.

**Type:** `boolean`  
**Default:** `false`

### `tags delete`

Delete a tag by its name or UUID. **This cannot be undone**, and it also removes all of the tag's assignments.

```bash
$ censys tags delete my-tag        # delete a tag by name (prompts for confirmation)
$ censys tags delete my-tag --yes  # delete without confirming
```

You are prompted to confirm before the tag is deleted. In a non-interactive terminal there is nobody to prompt, so `--yes` is required — without it the command fails rather than deleting silently.

#### Flags

**`--yes`, `-y`**: Skip the confirmation prompt.

**Type:** `boolean`  
**Default:** `false`

### `tags assign`

Assign a tag to one or more assets.

```bash
$ censys tags assign my-tag 8.8.8.8                          # one asset
$ censys tags assign my-tag 8.8.8.8 1.1.1.1                  # several assets
$ censys tags assign my-tag example.com:443                  # a web property
$ censys tags assign my-tag 8.8.8.8 example.com:443          # asset types can be mixed
$ censys tags assign my-tag --input-file assets.txt          # read assets from a file
$ cat assets.txt | censys tags assign my-tag --input-file -  # read assets from STDIN
$ censys tags assign my-tag --query 'host.services.port: 22' # bulk: every matching asset
```

Positional assets may be separated by spaces or commas (`8.8.8.8,1.1.1.1`); a file supplies one asset per line. Assets are validated before anything is sent — if one identifier is unrecognized, the whole call is rejected and no assignment is made.

Each asset is then assigned **independently, one request per asset**: if one fails the rest still proceed, and every per-asset outcome is reported. A run where some assets succeeded and some failed is a *partial success* — it prints a summary to stderr and still **exits 0**. Only a run where every asset failed exits non-zero.

Assigning a tag to an asset that already has it fails for that asset with an `already exists` error from the API.

Passing `--query` switches to bulk mode; see [Bulk Operations](#bulk-operations).

#### Flags

**`--input-file`, `-i`**: File to read the assets from, one per line, or `-` for STDIN. **Overrides** positional asset arguments — if both are given, the file wins.

**Type:** `string` (path, or `-`)  
**Default:** none

**`--query`**: A CenQL query selecting the assets to tag. Starts a bulk job instead of assigning explicit assets, and cannot be combined with explicit assets or `--input-file`.

**Type:** `string`  
**Default:** none

**`--max-assets`**: Cap the number of assets a bulk job tags. Requires `--query`. A single bulk job tags at most **100,000 assets**, so the effective cap is the smallest of this flag, that ceiling, and your plan's tag asset limit — see [Bulk Operations](#bulk-operations).

**Type:** `integer` (≥ 0; `0` means no explicit cap)  
**Default:** none

**`--wait`, `-w`**: Poll the bulk job until it reaches a final status. Requires `--query`.

**Type:** `boolean`  
**Default:** `false`

**`--timeout`**: How long to wait before giving up. Requires `--wait`. Use `0` for no limit.

**Type:** `string` (duration, e.g. `5m`, `1h`)  
**Default:** `30m`

**`--yes`, `-y`**: Skip the confirmation prompt. Requires `--query` — explicit assignment does not prompt, so `--yes` is rejected there rather than silently ignored.

**Type:** `boolean`  
**Default:** `false`

### `tags unassign`

Unassign a tag from one or more assets.

```bash
$ censys tags unassign my-tag 8.8.8.8                              # one asset
$ censys tags unassign my-tag 8.8.8.8 1.1.1.1                      # several assets
$ censys tags unassign my-tag --input-file assets.txt              # read assets from a file
$ censys tags unassign my-tag --all                                # bulk: every assignment
$ censys tags unassign my-tag --created-before 2026-01-01T00:00:00Z # bulk: a time window
```

Explicit unassignment mirrors `assign`: the same space- or comma-separated asset input, validated up front, then each asset unassigned independently with per-asset outcomes and partial-success semantics.

Unassigning an asset the tag is **not** assigned to is reported as a failure for that asset, not silently ignored. This is deliberate — it surfaces typos instead of reporting success for an asset you never touched.

Like `assign`, explicit unassignment never prompts — the assets were named on the command line. Only bulk mode confirms.

Passing `--all` or a time filter switches to bulk mode; see [Bulk Operations](#bulk-operations).

#### Flags

**`--input-file`, `-i`**: File to read the assets from, one per line, or `-` for STDIN. **Overrides** positional asset arguments.

**Type:** `string` (path, or `-`)  
**Default:** none

**`--all`**: Remove every one of the tag's assignments. Starts a bulk job. Cannot be combined with explicit assets, nor narrowed by a time filter.

**Type:** `boolean`  
**Default:** `false`

**`--created-before`**: Only unassign assignments created before this time. Starts a bulk job.

**Type:** `string` (RFC3339 timestamp)  
**Default:** none

**`--created-after`**: Only unassign assignments created after this time. Starts a bulk job.

**Type:** `string` (RFC3339 timestamp)  
**Default:** none

**`--wait`, `-w`**: Poll the bulk job until it reaches a final status. Requires `--all` or a time filter.

**Type:** `boolean`  
**Default:** `false`

**`--timeout`**: How long to wait before giving up. Requires `--wait`. Use `0` for no limit.

**Type:** `string` (duration, e.g. `5m`, `1h`)  
**Default:** `30m`

**`--yes`, `-y`**: Skip the confirmation prompt. Requires `--all` or a time filter — explicit unassignment does not prompt, so `--yes` is rejected there rather than silently ignored.

**Type:** `boolean`  
**Default:** `false`

### `tags assignments`

List the assets a tag is assigned to.

```bash
$ censys tags assignments my-tag                                    # list a tag's assignments
$ censys tags assignments my-tag --asset-type host                  # only host assignments
$ censys tags assignments my-tag --asset 8.8.8.8                    # check whether one asset is assigned
$ censys tags assignments my-tag --created-after 2025-01-01T00:00:00Z
$ censys tags assignments my-tag --max-pages -1                     # fetch every page
$ censys tags assignments my-tag --streaming                        # emit NDJSON as assignments are fetched
```

This is the one `tags` command that supports **streaming**. `--streaming` / `-S` is a [global flag](../GLOBAL_CONFIGURATION.md), so it does not appear in this command's own flag list, but it is supported here: each assignment is emitted as NDJSON as it is fetched instead of being collected and rendered at the end. It cannot be combined with `--output-format`.

#### Flags

**`--asset`**: Filter by a single asset (host IP, certificate SHA-256 fingerprint, or web property `hostname:port`). Exactly one asset — this is a filter, not a list.

**Type:** `string`  
**Default:** none

**`--asset-type`**: Filter by asset type.

**Type:** `string` (`host` | `web_property` | `certificate`)  
**Default:** none

**`--created-by`**: Filter by the UUID of the assignment's creator.

**Type:** `string` (UUID format)  
**Default:** none

**`--created-before`**: Only assignments created before this time.

**Type:** `string` (RFC3339 timestamp)  
**Default:** none

**`--created-after`**: Only assignments created after this time. Must be earlier than `--created-before` if both are given.

**Type:** `string` (RFC3339 timestamp)  
**Default:** none

**`--order-by`**: Sort order for the results. Note this is a **different set** from `tags list` — assignments sort only by creation time.

**Type:** `string` (`create_time_asc` | `create_time_desc`)  
**Default:** none (the API's own ordering)

**`--page-size`, `-n`**: Number of assignments to return per page. The API caps this at 1000.

**Type:** `integer` (1–1000)  
**Default:** `100`

**`--max-pages`, `-p`**: Maximum number of pages to fetch. Use `-1` to fetch all pages.

**Type:** `integer`  
**Default:** `1`

### `tags operations list`

List the asynchronous jobs created by bulk tag operations. Given a tag, only that tag's operations are listed; omit it to list operations across every tag in the organization.

```bash
$ censys tags operations list                        # every tag's operations
$ censys tags operations list my-tag                 # one tag's operations
$ censys tags operations list my-tag --status running # only operations still in flight
$ censys tags operations list --type bulk_delete     # only bulk unassign jobs
$ censys tags operations list --max-pages -1         # fetch every page
```

#### Flags

**`--status`**: Filter by operation status.

**Type:** `string` (`pending`, `running`, `succeeded`, `limit_reached`, `failed`, `cancelled`)  
**Default:** none

**`--type`**: Filter by operation type. `bulk_create` jobs come from `tags assign --query`; `bulk_delete` jobs come from bulk `tags unassign`.

**Type:** `string` (`bulk_create` | `bulk_delete`)  
**Default:** none

**`--order-by`**: Sort order for the results.

**Type:** `string` (`create_time_asc` | `create_time_desc`)  
**Default:** none (the API's own ordering)

**`--page-size`, `-n`**: Number of operations to return per page.

**Type:** `integer` (1–1000)  
**Default:** `100`

**`--max-pages`, `-p`**: Maximum number of pages to fetch. Use `-1` to fetch all pages.

**Type:** `integer`  
**Default:** `1`

### `tags operations get`

Retrieve a single bulk tag operation, by the tag it belongs to and the operation's UUID.

```bash
$ censys tags operations get my-tag <operation-id>                    # show current status
$ censys tags operations get my-tag <operation-id> --wait             # poll until it finishes
$ censys tags operations get my-tag <operation-id> --wait --timeout 5m # give up waiting after 5 minutes
```

Without `--wait` this is a plain read: it reports whatever status the operation currently has and **exits 0**, even for a failed operation — reading a failed job is itself a successful read. Only `--wait` maps a terminal status onto the exit code (see [Bulk Operations](#bulk-operations)).

Interrupting a wait (Ctrl-C) stops the polling, not the job. The command tells you so and prints the command to resume tracking.

#### Flags

**`--wait`, `-w`**: Poll until the operation reaches a final status.

**Type:** `boolean`  
**Default:** `false`

**`--timeout`**: How long to wait before giving up. Requires `--wait`. Use `0` for no limit.

**Type:** `string` (duration, e.g. `5m`, `1h`)  
**Default:** `30m`

### `tags operations cancel`

Cancel a running bulk tag operation.

```bash
$ censys tags operations cancel my-tag <operation-id>   # stop a running bulk job
```

Cancelling stops the job from processing any more assets. **It is not a rollback** — assignments the job has already made or removed stay as they are.

An operation that has already finished cannot be cancelled; the API rejects it with a `Tag operation not cancellable` conflict. A successful cancellation exits 0.

This command does not prompt: it only stops further processing, and the destructive step was the job it is stopping. There is no `--yes` flag.

#### Flags

Only the global flags and `--org-id`.

## Bulk Operations

Tagging by query, and untagging in bulk, are **asynchronous**. Rather than acting immediately, they submit a job and return the operation tracking it:

```bash
$ censys tags assign my-tag --query 'host.services.port: 22'    # bulk_create job
$ censys tags unassign my-tag --all                             # bulk_delete job
$ censys tags unassign my-tag --created-before 2026-01-01T00:00:00Z
```

**Entering bulk mode.** `assign` enters it only via `--query`; it is never inferred from a missing asset list. `unassign` enters it via `--all` **or** a time filter. `--all` means *every* assignment, so narrowing it with `--created-before`/`--created-after` contradicts itself and is rejected. In both commands, bulk mode cannot be combined with explicit assets or `--input-file`, and the bulk-only flags (`--max-assets`, `--wait`, `--timeout`) are rejected outside it rather than silently ignored.

**How many assets a job tags.** A single bulk job tags at most **100,000 assets**, however many the query matches. Three separate limits apply and the smallest wins: this fixed per-job ceiling, your plan's overall tag asset limit, and `--max-assets` if you set one. A job that stops at a limit finishes as `limit_reached` rather than `failed`, and reports how many assets it processed. To cover a query matching more than the ceiling, split it into narrower queries and submit each as its own job.

**Confirmation.** Bulk operations always prompt before submitting, unless `--yes` is set. In a non-interactive terminal `--yes` is required — the prompt is gated before the job is submitted, so a script without it fails instead of launching a large job silently.

**Tracking.** Without `--wait` the command prints the operation and the command to track it. With `--wait` it polls until the operation reaches a final status, reporting progress as it goes.

**Waiting and exit codes.** `--wait` requires bulk mode, and `--timeout` requires `--wait`. `--timeout 0` means *no limit* (matching the global `--timeout-http`); a negative duration is rejected. When a wait ends, the final status maps onto the exit code:

| Final status                   | Result                                                          |
| ------------------------------ | --------------------------------------------------------------- |
| `succeeded`                    | exit 0                                                            |
| `limit_reached`                | exit 0, with a warning naming how many assets were processed     |
| `failed`, `cancelled`          | exit 1                                                            |
| timeout expired while running  | exit 124 — the job continues server-side                          |
| interrupted (Ctrl-C)           | exit 130 — the job continues server-side                          |

A capped run (`limit_reached`) still did its work, so it warns rather than failing. Interrupting or timing out stops only the *polling*; the command prints the `tags operations get` command to resume tracking.

Use [`tags operations`](#tags-operations-list) to list, inspect, and cancel these jobs afterwards.

## Search Index Lag

Tag assignments are not reflected in search immediately. After a successful `assign` or `unassign`, it may take a few minutes for the change to appear in — or disappear from — `tags:` search results. The commands print a note to that effect on success (suppressed by `--quiet`).

This affects search only. `tags assignments` and `tags get --asset-count` read the assignments directly and reflect changes right away.

## Output Formats

All `tags` commands default to **`short`** output. Override with `--output-format` (or `-O`).

**Default:** `short`  
**Supported formats:** `short`, `json`, `yaml`, `tree`

- **`short`** — human-readable: a styled table for the list-shaped commands (`list`, `assignments`, `operations list`, and the per-asset results of `assign`/`unassign`), a detail view for the single-record commands
- **`json`** — structured JSON
- **`yaml`** — structured YAML
- **`tree`** — hierarchical tree view (interactive; requires a terminal)

Templates (`-O template`) are not supported for `tags`.

> **Scripting note:** the `assign` and `unassign` payload **shape depends on the mode**. With explicit assets they emit an **array of per-asset results**; in bulk mode they emit a **single operation object**, identical in shape to `tags operations get`. A script consuming `-O json` from these two commands must branch on which mode it invoked.

## Exit Codes

| Code | Meaning                                                                     |
| ---- | --------------------------------------------------------------------------- |
| 0    | Success, including a partial success where some assets failed                |
| 1    | API error, missing credentials, or a waited-on operation that ended `failed`/`cancelled` |
| 2    | Usage or input error — an invalid flag value, an unknown asset, a rejected flag combination |
| 124  | Timed out                                                                    |
| 130  | Interrupted                                                                  |

Partial failures are reported on stderr and do not change the exit code; check the per-asset results in the output to see which assets failed.
