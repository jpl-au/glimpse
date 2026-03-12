# Glimpse

A terminal-based JSON inspector with inline image preview.

Glimpse renders a split-view TUI showing formatted JSON on the left and an
embedded image preview on the right. If the JSON contains base64-encoded image
data (PNG, JPEG, GIF, WEBP, TIFF), it is automatically detected and displayed.

A persistent filter bar at the top of the JSON pane accepts free-text search,
matching keys and values case-insensitively. Results update live as you type.

## Install

```sh
go install github.com/jpl-au/glimpse@latest
```

## Usage

```sh
glimpse <json-file>
```

Glimpse also reads from stdin, so you can pipe JSON from any source:

```sh
cat data.json | glimpse -
psql -t -c "SELECT data FROM docs WHERE id = 42" mydb | glimpse -
curl -s https://api.example.com/data | glimpse -
```

## Controls

| Key        | Action                                  |
|------------|-----------------------------------------|
| type       | Search keys and values in the JSON      |
| `esc`      | Clear filter, or quit if filter is empty|
| `ctrl+c`   | Quit                                   |
| `tab`      | Switch pane                            |
| `up` / `down` | Scroll JSON or cycle images         |
| `ctrl+p`   | Full-screen HD image                   |

## License

[MIT](LICENSE)
