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

Show which credential is currently used to authenticate API requests: the OAuth login or a stored personal access token, whichever was most recently activated. For OAuth logins it shows the logged-in account (email) and whether the session is scoped to an organization (by name) or to your free account. The organization name is resolved once at login and stored with the session, so `auth status` needs no network access.

## Organization context

An OAuth login is **locked to a single context**, chosen when you log in: either **one organization** or your **free account**. The choice is embedded in the OAuth access token itself (the consent screen is where you pick the organization, or skip it for your free account), and it is **strict in both directions**:

- A session logged in to an **organization** acts only on that organization. It **cannot** access free-user endpoints such as `censys credits` (free user credits).
- A session logged in to your **free account** acts only on your free account. It **cannot** access any organization — even one you belong to, and even one you join later — so `censys org ...` and `censys enrich` are unavailable.

Because the session carries its own context, you do **not** set or pass an organization ID with an OAuth login: any stored `censys config org-id` value and the `--org-id` flag are ignored while you are logged in via OAuth. The Platform API resolves (and enforces) the organization from the token.

### Switching organizations or your free account

There is no way to switch context within a session, and only one OAuth session exists at a time. To move between organizations, or between an organization and your free account, **log out and log in again**:

```bash
$ censys auth logout
$ censys auth login   # pick the organization (or skip it for your free account) on the consent screen
```

`cencli` fails fast with a clear message when you run a command that the current session isn't scoped for (for example `censys credits` while logged in to an organization), so you don't have to interpret a raw API error.

### Personal access tokens differ

Personal access tokens are **not** organization-scoped: a PAT can act across any organization you belong to as well as your free account. With a PAT you may configure an organization ID (`censys config org-id`) or pass `--org-id` per command; if none is configured, requests use your free-user wallet by default.

## Credential precedence

`cencli` uses whichever credential was most recently activated:

- `censys auth login` makes the OAuth login active.
- `censys config auth activate <id>` (or adding a new personal access token) makes that token active.
- `censys auth logout` removes the OAuth login entirely.

Manage personal access tokens with `censys config auth` (see the [config command docs](./CONFIG.md#config-auth)).
