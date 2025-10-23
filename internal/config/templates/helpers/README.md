# Template Helpers

Handlebars helpers for CLI templates.

## Available Helpers

### Color Helpers
Apply colors to text. Available colors: `red`, `blue`, `orange`, `yellow`

```handlebars
{{red "Error message"}}
{{blue "Info message"}}
{{orange "Warning message"}}
{{yellow "Highlight"}}
```

### `capitalize`
Capitalizes letters in a string based on the mode.

**Arguments:**
- `value` (required): The string to capitalize
- `mode` (required): Either `"first"` or `"all"`
  - `"first"`: Capitalizes only the first letter of the string
  - `"all"`: Uppercases all letters in the string

```handlebars
{{capitalize "hello world" "first"}} {{!-- "Hello world" --}}
{{capitalize "hello world" "all"}}   {{!-- "HELLO WORLD" --}}
```

### `length`
Returns the length of arrays, slices, maps, or strings.

```handlebars
{{length myArray}}  {{!-- "5" --}}
```

### `join`
Joins a list of strings with a delimiter.

```handlebars
{{join tags ", "}}  {{!-- "go, cli, tool" --}}
{{join ports " | "}}  {{!-- "80 | 443 | 8080" --}}
```

### `pluck`
Extracts values from an array of objects/maps by key and filters out falsy values (nil, empty string, false, 0).

**Arguments:**
- `array` (required): The array of objects/maps to extract from
- `key` (required): The key/field name to extract

```handlebars
{{join (pluck users "name") ", "}}  {{!-- "Alice, Bob, Charlie" --}}
{{join (pluck users "email") ", "}}  {{!-- filters out empty/nil emails --}}
```

The helper automatically filters out falsy values, so only valid values are included in the result.

### `concat`
Concatenates an array of strings, automatically filtering out empty strings.

**Arguments:**
- `array` (required): The array of strings to concatenate

```handlebars
{{concat pathParts}}  {{!-- "home/user/documents" --}}
{{concat urlParts}}   {{!-- "https://example.com:443/api/v1" --}}
```

The helper automatically filters out empty strings, so only non-empty values are included in the result.

### Comparison Helpers
Compare values: `lt` (less than), `gt` (greater than), `eq` (equal).

```handlebars
{{#if (lt count 10)}}Less than 10{{/if}}
{{#if (gt score 100)}}Over 100{{/if}}
{{#if (eq status "active")}}Active{{/if}}
```

### `platform_lookup_url`
Generates Censys platform lookup URLs.

```handlebars
{{platform_lookup_url "host" "<ip address>"}}
{{platform_lookup_url "certificate" "<certificate sha-256>"}}
{{platform_lookup_url "webproperty" "<hostname:port>"}}
```

