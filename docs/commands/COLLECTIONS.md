# Collections Command

The `collections` command manages collections for your organization. A collection is a saved CenQL query whose matching assets Censys keeps up to date. Use `censys search --collection-id` or `censys aggregate --collection-id` to query inside a collection.

Running `censys collections` without a subcommand prints help.

## Usage

```bash
$ censys collections list                                # list all collections
$ censys collections get <collection-id>                 # show one collection
$ censys collections create my-collection --query "..."  # create a collection
$ censys collections update <collection-id> --name new   # change a collection
$ censys collections delete <collection-id>              # delete a collection
```

## Organization Context

Every `collections` subcommand accepts **`--org-id`, `-o`** (type `string`, UUID format), and every one **requires** an organization. Collections have no free-account fallback.

- **Personal access token** — the stored organization ID by default, or `--org-id` per subcommand. To store a default, run `censys config org-id add` (see the [config command docs](./CONFIG.md)). With neither, the command fails before sending anything (exit 2).
- **OAuth login (`censys auth login`)** — the organization is fixed by what that login was authorized for. Stored organization IDs are ignored and `--org-id` fails with an error; run `censys auth logout` and log in again to target a different organization. A login authorized for your free account cannot use collections (exit 1). See [Organization context](AUTH.md#organization-context).

To see global flags and how they affect these commands, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

## Collection Identifiers

Collections are identified by **UUID only**. `collections get`, `update`, and `delete` all take a `<collection-id>` positional argument that must parse as a UUID; anything else is rejected before a request is made. Use `censys collections list` to find a collection's ID.

## Commands

### `collections list`

List collections in your organization.

```bash
$ censys collections list                             # list all collections
$ censys collections list --status active             # list only active collections
$ censys collections list --status active,paused      # list active and paused collections
$ censys collections list --max-pages -1              # fetch every page
$ censys collections list --output-format json        # output as JSON
```

Results are paginated. By default `list` fetches one page (100 collections); use `--max-pages` to fetch more, or `-1` for all pages. The API reports no total count, so the `short` output header reads `Collections (N)` for however many were fetched, never a total. When more collections exist beyond what was fetched, the command prints a note on stderr — pass `--quiet` to suppress it.

#### Flags

**`--status`**: Filter by status. Repeatable, or comma-separated, to match more than one status.

**Type:** `strings` (`populating`, `active`, `paused`, `archived`)  
**Default:** none (no status filter)

**`--page-size`, `-n`**: Number of collections to return per page.

**Type:** `integer`  
**Default:** `100`

**`--max-pages`, `-p`**: Maximum number of pages to fetch. Use `-1` to fetch all pages.

**Type:** `integer`  
**Default:** `1`

#### Sample Output

```console
$ censys collections list

Collections (2)

ID                                     Name         Status   Assets   +24h   -24h   Created At      

550e8400-e29b-41d4-a716-446655440000 | ssh-hosts  | active | 1204   | 12   | 3    | 2026-09-01 10:15
6ba7b810-9dad-11d1-80b4-00c04fd430c8 | rdp-review | paused | 88     | 0    | 0    | 2026-09-10 14:02
```

```json
$ censys collections list --output-format json

[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "ssh-hosts",
    "description": "All SSH hosts",
    "query": "host.services.protocol=SSH",
    "status": "active",
    "total_assets": 1204,
    "added_assets_24_hours": 12,
    "removed_assets_24_hours": 3,
    "created_by": "6f985c2a-daa6-4a37-b666-03b4aecd8a88",
    "create_time": "2026-09-01T10:15:00Z"
  }
]
```

### `collections get`

Retrieve a single collection by its UUID.

```bash
$ censys collections get <collection-id>                      # get a collection
$ censys collections get <collection-id> --output-format json # output as JSON
```

#### Flags

Only the global flags and `--org-id`.

#### Sample Output

```console
$ censys collections get 550e8400-e29b-41d4-a716-446655440000

━━━ Collection ━━━

  Name:         ssh-hosts
  ID:           550e8400-e29b-41d4-a716-446655440000
  Description:  All SSH hosts
  Query:        host.services.protocol=SSH
  Status:       active
  Assets:       1204
  Added 24h:    12
  Removed 24h:  3
  Created By:   6f985c2a-daa6-4a37-b666-03b4aecd8a88
  Created At:   2026-09-01 10:15:00 UTC
```

```json
$ censys collections get 550e8400-e29b-41d4-a716-446655440000 --output-format json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "ssh-hosts",
  "description": "All SSH hosts",
  "query": "host.services.protocol=SSH",
  "status": "active",
  "total_assets": 1204,
  "added_assets_24_hours": 12,
  "removed_assets_24_hours": 3,
  "created_by": "6f985c2a-daa6-4a37-b666-03b4aecd8a88",
  "create_time": "2026-09-01T10:15:00Z"
}
```

### `collections create`

Create a new collection from a CenQL query.

```bash
$ censys collections create ssh-hosts --query "host.services.protocol=SSH"                               # create a collection
$ censys collections create ssh-hosts --query "host.services.protocol=SSH" --description "All SSH hosts" # create a collection with a description
```

Censys populates the collection in the background, so a newly created collection can show `populating` until the first build finishes. On success, the command prints a stderr hint showing how to search the new collection, e.g. `censys search --collection-id <collection-id> "host.services.protocol=SSH"`.

#### Flags

**`--query`**: CenQL query that selects the collection's assets. Required.

**Type:** `string`  
**Default:** none

**`--description`**: A human-readable description of the collection.

**Type:** `string`  
**Default:** none

#### Sample Output

```console
$ censys collections create ssh-hosts --query "host.services.protocol=SSH"

━━━ Collection Created ━━━

  Name:         ssh-hosts
  ID:           550e8400-e29b-41d4-a716-446655440000
  Description:  -
  Query:        host.services.protocol=SSH
  Status:       populating
  Assets:       0
  Added 24h:    0
  Removed 24h:  0
  Created By:   6f985c2a-daa6-4a37-b666-03b4aecd8a88
  Created At:   2026-09-24 09:00:00 UTC

Search this collection: censys search --collection-id 550e8400-e29b-41d4-a716-446655440000 'host.services.protocol=SSH'
```

```json
$ censys collections create ssh-hosts --query "host.services.protocol=SSH" --output-format json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "ssh-hosts",
  "description": "",
  "query": "host.services.protocol=SSH",
  "status": "populating",
  "total_assets": 0,
  "added_assets_24_hours": 0,
  "removed_assets_24_hours": 0,
  "created_by": "6f985c2a-daa6-4a37-b666-03b4aecd8a88",
  "create_time": "2026-09-24T09:00:00Z"
}
```

When the organization is at its collection limit, the API refuses the create with a 412 and the command reports how many collections count toward it:

```console
$ censys collections create ssh-hosts --query "host.services.protocol=SSH"

[Collection Limit Reached]
your organization has reached its collection limit (12 collection(s) count toward it; archived collections do not). Delete one with `censys collections delete <collection-id>`, or contact your Censys account team for more
```

### `collections update`

Update an existing collection by its UUID. **At least one mutation flag is required** — an update with nothing to change is rejected rather than sent.

```bash
$ censys collections update <collection-id> --name renamed                           # rename a collection
$ censys collections update <collection-id> --query "host.services.protocol=RDP"     # change the query
$ censys collections update <collection-id> --description "Hosts to review"          # set a description
$ censys collections update <collection-id> --clear-description                      # remove the description
```

The API replaces the whole collection on every update, so this command first reads the collection, then sends the full record back with only the requested fields changed — every value you do not change is kept as-is. This means a change someone else makes to the collection between the read and the update is overwritten.

#### Flags

**`--name`**: A new name for the collection.

**Type:** `string`  
**Default:** none (name unchanged)

**`--query`**: A new CenQL query for the collection.

**Type:** `string`  
**Default:** none (query unchanged)

**`--description`**: A new description for the collection. Cannot be combined with `--clear-description`.

**Type:** `string`  
**Default:** none (description unchanged)

**`--clear-description`**: Remove the collection's description. Cannot be combined with `--description`.

**Type:** `boolean`  
**Default:** `false`

#### Sample Output

```console
$ censys collections update 550e8400-e29b-41d4-a716-446655440000 --name renamed-hosts

━━━ Collection Updated ━━━

  Name:         renamed-hosts
  ID:           550e8400-e29b-41d4-a716-446655440000
  Description:  All SSH hosts
  Query:        host.services.protocol=SSH
  Status:       active
  Assets:       1204
  Added 24h:    12
  Removed 24h:  3
  Created By:   6f985c2a-daa6-4a37-b666-03b4aecd8a88
  Created At:   2026-09-01 10:15:00 UTC
```

```json
$ censys collections update 550e8400-e29b-41d4-a716-446655440000 --name renamed-hosts --output-format json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "renamed-hosts",
  "description": "All SSH hosts",
  "query": "host.services.protocol=SSH",
  "status": "active",
  "total_assets": 1204,
  "added_assets_24_hours": 12,
  "removed_assets_24_hours": 3,
  "created_by": "6f985c2a-daa6-4a37-b666-03b4aecd8a88",
  "create_time": "2026-09-01T10:15:00Z"
}
```

### `collections delete`

Delete a collection by its UUID. **This cannot be undone.**

```bash
$ censys collections delete <collection-id>        # delete a collection (prompts for confirmation)
$ censys collections delete <collection-id> --yes  # delete without confirming
```

You are prompted to confirm before the collection is deleted. In a non-interactive terminal there is nobody to prompt, so `--yes` is required — without it the command fails rather than deleting silently.

#### Flags

**`--yes`, `-y`**: Skip the confirmation prompt.

**Type:** `boolean`  
**Default:** `false`

#### Sample Output

```console
$ censys collections delete 550e8400-e29b-41d4-a716-446655440000 --yes

Collection "550e8400-e29b-41d4-a716-446655440000" deleted.
```

```json
$ censys collections delete 550e8400-e29b-41d4-a716-446655440000 --yes --output-format json

{
  "collection": "550e8400-e29b-41d4-a716-446655440000",
  "deleted": true
}
```

## Data Availability

- Collection limits depend on your organization's plan. The API enforces them when you create a collection. It does not expose the limit or your remaining allowance, so the CLI cannot show how many more collections you can create, or check before it creates one.
- When the organization is at its limit, `collections create` fails with "Collection Limit Reached". The message shows how many collections count toward the limit. Archived collections do not count.
- There is no pre-check: the API is the only source of the limit.
- The list endpoint does not return a total count. `collections list` reports how many collections it fetched, and prints a note on stderr when more pages exist.
- See the [Censys plan documentation](https://docs.censys.com/docs/platform-collections) for collection limits by plan.

## Output Formats

All `collections` commands default to **`short`** output. Override with `--output-format` (or `-O`).

**Default:** `short`  
**Supported formats:** `short`, `json`, `yaml`, `tree`

- **`short`** — human-readable: a styled table for `list`, a detail view for `get`, `create`, and `update`, and a one-line confirmation for `delete`
- **`json`** — structured JSON
- **`yaml`** — structured YAML
- **`tree`** — hierarchical tree view (interactive; requires a terminal)

Templates (`-O template`) are not supported for `collections`.

## Exit Codes

| Code | Meaning                                                                     |
| ---- | --------------------------------------------------------------------------- |
| 0    | Success                                                                      |
| 1    | API error, missing or invalid credentials, the collection limit was reached, or the API returned no usable collection to update |
| 2    | Usage or input error — an invalid ID, name, or query, an unsupported `--status` value, invalid pagination flags, a rejected flag combination, nothing to update, or missing confirmation in a non-interactive terminal |
| 124  | Timed out                                                                    |
| 130  | Interrupted                                                                  |

## Related Commands

- [`search`](SEARCH.md) — query within a collection with `censys search --collection-id <collection-id> "<query>"`
- [`aggregate`](AGGREGATE.md) — aggregate within a collection with `censys aggregate --collection-id <collection-id> "<query>" "<field>"`
