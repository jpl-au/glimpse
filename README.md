# Glimpse

A terminal-based JSON inspector with inline image preview.

Glimpse renders a split-view TUI showing formatted JSON on the left and an
embedded image preview on the right. If the JSON contains base64-encoded image
data (PNG, JPEG, GIF, WEBP, TIFF), it is automatically detected and displayed.

Press `i` to open a full-screen high-quality image view using kitty or sixel
graphics protocols.

## Install

```sh
go install github.com/jpl-au/glimpse@latest
```

## Usage

```sh
glimpse <json-file>
```

## Controls

| Key       | Action              |
|-----------|---------------------|
| `tab`     | Switch pane         |
| `↑` / `↓` | Scroll             |
| `i`       | Full-screen HD image |
| `esc`     | Return / quit       |
| `q`       | Quit                |

## License

[MIT](LICENSE)
