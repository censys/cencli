# Tags Command

The `tags` command manages tags and tag assignments for your organization. Tags let you label and organize assets (hosts, certificates, and web properties) for tracking and filtering.

Running `censys tags` without a subcommand prints help.

> **Note:** This is the first slice of the tags feature. Today it ships `tags list`; the remaining commands (`get`, `create`, `update`, `delete`, `assign`, `unassign`, `assignments`, `operations`) are being added incrementally. This document describes only what currently ships.

By default, these commands use your stored organization ID. If no organization ID is stored, or you want to query a different organization, use the `--org-id` flag. To set a default, run `censys config org-id set <org-id>` (see the [config command docs](./CONFIG.md)).

## `tags list`

List all tags in your organization.

```bash
$ censys tags list                        # list all tags
$ censys tags list --privacy shared       # only shared tags
$ censys tags list --name my-tag          # filter by exact name
$ censys tags list --order-by name_desc   # sort by name, descending
$ censys tags list --output-format json   # output as JSON
```

Results are paginated. By default only the first page is fetched; use `--max-pages` to fetch more (or `-1` for all pages).

### Flags

To see global flags and how they affect this command, see the [global configuration docs](../GLOBAL_CONFIGURATION.md).

#### `--privacy`

Filter tags by privacy setting. `private` tags are visible to and editable by organization admins only; `shared` tags are visible to all organization members.

**Type:** `string` (`private` | `shared`)
**Default:** none (no privacy filter)

#### `--name`

Filter tags by exact name.

**Type:** `string`
**Default:** none

#### `--created-by`

Filter tags by the user ID of the user who created them.

**Type:** `string`
**Default:** none

#### `--order-by`

Sort order for the results.

**Type:** `string` (`name_asc`, `name_desc`, `created_at_asc`, `created_at_desc`, `updated_at_asc`, `updated_at_desc`)
**Default:** `name_asc`

#### `--page-size`, `-n`

Number of tags to return per page.

**Type:** `integer`
**Default:** `100`

#### `--max-pages`, `-p`

Maximum number of pages to fetch. Use `-1` to fetch all pages.

**Type:** `integer`
**Default:** `1`

#### `--org-id`, `-o`

Organization ID to use for the request, overriding the configured default.

**Type:** `string` (UUID format)
**Default:** uses the configured organization ID

## Output Formats

The `tags list` command defaults to **`short`** output — a styled table of tags. Override with `--output-format` (or `-O`).

**Default:** `short`
**Supported formats:** `short`, `json`, `yaml`, `tree`

- **`short`** — a concise table (name, privacy, description, creator, created-at)
- **`json`** — structured JSON
- **`yaml`** — structured YAML
- **`tree`** — hierarchical tree view (interactive; requires a terminal)
