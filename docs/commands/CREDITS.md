# Credits Command

The `credits` command displays credit details for your free user Censys account.

## Usage

```bash
$ censys credits  # show free user credits
```

## Description

This command shows your personal free user credit balance and usage information. Free user credits are associated with your individual Censys account, not an organization.

**Note:** This command only shows free user credits. If you want to see organization credits for a paid account, use [`censys org credits`](ORG.md#org-credits) instead.

**OAuth logins:** If you logged in with `censys auth login` and selected an **organization**, your session is locked to that organization and **cannot** access free user credits — this command fails with a message directing you to `censys org credits`. To view free user credits, log out and log in again without selecting an organization (see [Organization context](AUTH.md#organization-context)).

## Output Formats

The `credits` command defaults to **`short`** output format, which displays results in a human-readable format. You can override this with the `--output-format` flag (or `-O`).

**Default:** `short`  
**Supported formats:** `json`, `yaml`, `tree`, `short`

### Examples

```bash
# Default: human-readable output
$ censys credits

# JSON output
$ censys credits --output-format json
$ censys credits -O json

# YAML output
$ censys credits --output-format yaml
```

## Free User vs Organization Credits

| Command | Description |
|---------|-------------|
| `censys credits` | Shows free user credits for your personal account |
| `censys org credits` | Shows organization credits for paid accounts |

If you have a paid organization account and want to check your organization's credit balance, use `censys org credits` instead.
