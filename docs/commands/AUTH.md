# Auth Command

The `auth` command authenticates you with the Censys Platform using your web browser (OAuth2 authorization code flow with PKCE). It is an alternative to configuring a personal access token with `censys config auth add`.

## Usage

```bash
$ censys auth login    # log in via your browser
$ censys auth status   # show which credential is used for API requests
$ censys auth logout   # revoke and remove the browser login
```

## Subcommands

### `auth login`

Log in to the Censys Platform via your browser.

```bash
$ censys auth login
Your browser has been opened to visit:

    https://oauth2.censys.io/oauth2/auth?client_id=censys-cencli&...

Waiting for login to complete...

✅ You are now logged in as [you@example.com]
```

The command starts a temporary listener on `127.0.0.1:5555`, opens your browser to the Censys authorization server, and captures the resulting tokens once you complete the login. Your **email address is shown** once login succeeds. Tokens are stored in the local credential database (`~/.config/cencli/cencli.db`) and access tokens are **refreshed automatically** when they expire — you stay logged in until you run `censys auth logout` (or the refresh token is revoked server-side).

The login waits up to 5 minutes for the browser flow to complete. Port `5555` must be free while logging in.

#### Flags for `auth login`

**`--no-browser`**: Print the login URL instead of opening a browser. Useful over SSH — note the redirect still targets `127.0.0.1:5555`, so the browser must run on (or forward to) the machine running `cencli`.

### `auth logout`

Revoke the OAuth2 session obtained via `censys auth login` (best effort) and remove it from local storage. Stored personal access tokens are not affected; if one exists it becomes the active credential again.

### `auth status`

Show which credential is currently used to authenticate API requests: the OAuth login or a stored personal access token, whichever was most recently activated. For OAuth logins it shows the logged-in account (email).

## Organization context

You do **not** need to set an organization ID to use an OAuth login. When you log in with `censys auth login`, your organization is **embedded in the OAuth access token** itself — the Platform API resolves your organization from the token, so there is nothing to provide on the command line or store beforehand.

This differs from personal access tokens, which are not organization-scoped: with a PAT you may still want to configure an organization ID (`censys config org-id`) or pass `--org-id` per command. If no organization is configured for a PAT, requests use your free-user wallet by default.

## Credential precedence

`cencli` uses whichever credential was most recently activated:

- `censys auth login` makes the OAuth login active.
- `censys config auth activate <id>` (or adding a new personal access token) makes that token active.
- `censys auth logout` removes the OAuth login entirely.

Manage personal access tokens with `censys config auth` (see the [config command docs](./CONFIG.md#config-auth)).
