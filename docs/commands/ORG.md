# Org Command

The `org` command allows you to manage and view organization details including credits, members, and organization information.

## Usage

```bash
$ censys org credits      # display credit details for your organization
$ censys org details      # display organization details
$ censys org members      # list organization members
```

By default, these commands use your stored organization ID. If no organization ID is stored, or you want to query a different organization, use the `--org-id` flag on each subcommand.

To set your default organization ID, run: `censys config org-id add`

## Subcommands

### `org credits`

Display credit details for your organization, including credit balance, auto-replenish configuration, and any credit expirations.

```bash
$ censys org credits                      # show credits for your stored organization
$ censys org credits --org-id <uuid>      # show credits for a specific organization
$ censys org credits --output-format json # output as JSON
```

#### Flags

**`--org-id`, `-o`**: Override the configured organization ID.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID

### `org details`

Display details about your organization, including name, ID, creation date, and member counts.

```bash
$ censys org details                      # show details for your stored organization
$ censys org details --org-id <uuid>      # show details for a specific organization
$ censys org details --output-format json # output as JSON
```

#### Flags

**`--org-id`, `-o`**: Override the configured organization ID.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID

### `org members`

List all members in your organization, including their email, name, roles, and last login time.

```bash
$ censys org members                      # list members for your stored organization
$ censys org members --interactive        # list members in an interactive table
$ censys org members --org-id <uuid>      # list members for a specific organization
$ censys org members --output-format json # output as JSON
```

#### Flags

**`--org-id`, `-o`**: Override the configured organization ID.

**Type:** `string` (UUID format)  
**Default:** Uses the configured organization ID

**`--interactive`, `-i`**: Display results in an interactive table (TUI) that allows you to navigate through the member list.

**Type:** `boolean`  
**Default:** `false`

## Output Formats

The `org` subcommands default to **`short`** output format, which displays results in a human-readable format. You can override this with the `--output-format` flag (or `-O`).

**Default:** `short`  
**Supported formats:** `json`, `yaml`, `tree`, `short`

## Note on Free User Credits

The `org credits` command shows organization credits for paid accounts. If you want to see your free user credits instead, use the [`censys credits`](CREDITS.md) command.
