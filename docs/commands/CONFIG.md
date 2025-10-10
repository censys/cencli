# Config Command

The `config` command manages authentication tokens, organization IDs, and other configuration settings for the CLI. These values are stored locally and attached to API requests automatically.

## Usage

```bash
$ censys config auth          # manage personal access tokens (interactive TUI)
$ censys config org-id        # manage organization IDs (interactive TUI)
$ censys config print         # print current configuration
```

## Subcommands

### `config auth`

Manage personal access tokens (PATs) used for authentication with the Censys API. The active token is automatically attached to all API requests.

Use `censys config auth add` to open an interactive prompt to add a token. Use `censys config auth` to view all stored tokens, activate a different token, or delete tokens.

There also exists a non-interactive mode for adding tokens, where you can provide the token value, name, and optionally activate the token.

```bash
$ censys config auth -a                                    # list all tokens
$ censys config auth add --value "your-token" --name "my-token"   # add a token (non-interactive)
$ censys config auth add --value-file token.txt --name "my-token" # add from file
$ censys config auth activate <id>                          # activate a specific token by ID
$ censys config auth delete <id>                            # delete a token by ID
```

### `config org-id`

Manage organization IDs used for API requests. The active organization ID is automatically attached to requests that support organization-scoped operations.

Use `censys config org-id add` to open an interactive prompt to add an organization ID. Use `censys config org-id` to view all stored organization IDs, activate a different one, or delete them.

There also exists a non-interactive mode for adding organization IDs, where you can provide the organization ID value, name, and optionally activate the organization ID.

```bash
$ censys config org-id -a                                              # list all org IDs
$ censys config org-id add --value "uuid" --name "production"           # add an org ID (non-interactive)
$ censys config org-id add --value-file orgid.txt --name "production"   # add from file
$ censys config org-id activate <id>                                    # activate a specific org ID by ID
$ censys config org-id delete <id>                                      # delete an org ID by ID
```


**Note:** If no organization ID is configured, requests will use your free-user wallet by default. You can also override the active organization ID for individual commands using the `--org-id` flag.

### `config print`

Print the current configuration in YAML format, including all settings from your configuration file. See the [global configuration docs](../GLOBAL_CONFIGURATION.md) for details on all available configuration options.

