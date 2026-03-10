# Glimpse

A terminal-based JSON inspector with inline image preview and gjson filtering.

Glimpse renders a split-view TUI showing formatted JSON on the left and an
embedded image preview on the right. If the JSON contains base64-encoded image
data (PNG, JPEG, GIF, WEBP, TIFF), it is automatically detected and displayed.

A persistent filter bar at the top of the JSON pane accepts
[gjson](https://github.com/tidwall/gjson) path expressions, letting you drill
into large documents without leaving the TUI. Results update live as you type.

## Install

```sh
go install github.com/jpl-au/glimpse@latest
```

## Usage

```sh
glimpse <json-file>
```

## Controls

| Key        | Action                                      |
|------------|---------------------------------------------|
| type       | Enter a gjson filter path in the filter bar  |
| `esc`      | Clear filter, or quit if filter is empty     |
| `ctrl+c`   | Quit                                        |
| `tab`      | Switch pane                                 |
| `↑` / `↓`  | Scroll                                     |
| `ctrl+p`   | Full-screen HD image                        |

## Filter Examples

| Path                      | Description                          |
|---------------------------|--------------------------------------|
| `name`                    | Select the `name` field              |
| `users.#.email`           | All email addresses in `users` array |
| `users.#(age>30)#.name`   | Names of users older than 30         |
| `config.database`         | Nested object selection              |
| `items.0`                 | First element of `items` array       |

See the [gjson syntax documentation](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)
for the full query language.

## License

[MIT](LICENSE)
