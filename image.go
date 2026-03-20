package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi/sixel"
	"github.com/dolmen-go/kittyimg"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	// cellPixelWidth assumes 8px per cell, matching the typical VT100 cell
	// geometry used by kitty and sixel renderers.
	cellPixelWidth = 8

	// cellPixelHeight assumes 16px per cell (2:1 height-to-width ratio),
	// matching most monospaced terminal fonts.
	cellPixelHeight = 16

	// pixelsPerRow is 2 because half-block characters (U+2580 "▀") encode
	// two vertical pixels per terminal row via foreground/background colours.
	pixelsPerRow = 2
)

// imagePlaceholder replaces base64 image data in the parsed JSON tree so
// that formatting and search operate on clean, human-readable values.
const imagePlaceholder = "(base64 image data)"

// imageEntry holds a decoded image and its JSON key name.
type imageEntry struct {
	key string
	img image.Image
}

// findImages walks data, decoding any base64 image strings it finds.
// Matched values are replaced in-place with imagePlaceholder so that
// subsequent formatting and searching skip them automatically. Map keys
// are visited in sorted order for deterministic results.
func findImages(data any) []imageEntry {
	var entries []imageEntry
	collectImages(data, "", &entries)
	return entries
}

func collectImages(data any, key string, entries *[]imageEntry) {
	switch v := data.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s, ok := v[k].(string)
			if !ok || !isImageString(s) {
				collectImages(v[k], k, entries)
				continue
			}
			v[k] = imagePlaceholder
			img, err := decodeImage(s)
			if err != nil {
				slog.Warn("skipping image", "key", k, "err", err)
				continue
			}
			*entries = append(*entries, imageEntry{key: k, img: img})
		}
	case []any:
		for i, item := range v {
			s, ok := item.(string)
			if !ok || !isImageString(s) {
				collectImages(item, key, entries)
				continue
			}
			v[i] = imagePlaceholder
			img, err := decodeImage(s)
			if err != nil {
				slog.Warn("skipping image", "key", key, "err", err)
				continue
			}
			name := key
			if name == "" {
				name = "image"
			}
			*entries = append(*entries, imageEntry{key: name, img: img})
		}
	}
}

// decodeImage decodes a base64-encoded image string (with or without a
// data-URI prefix) into a Go image.Image.
func decodeImage(data string) (image.Image, error) {
	clean := data
	if strings.HasPrefix(data, "data:image/") {
		parts := strings.SplitN(data, ",", 2)
		if len(parts) == 2 {
			clean = parts[1]
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}
	return img, nil
}

// renderPreview renders an image as colored text using half-block characters
// (U+2580 "▀"). Each character cell encodes two vertical pixels via foreground
// (top) and background (bottom) colors, doubling vertical resolution. The
// image is aspect-ratio-preserved and centered within the given area.
func renderPreview(img image.Image, cols, rows int) string {
	bounds := img.Bounds()
	imgW := float64(bounds.Dx())
	imgH := float64(bounds.Dy())

	// Each terminal cell is roughly twice as tall as it is wide. With
	// half-blocks we get pixelsPerRow pixels per row, so the effective
	// pixel grid is cols wide × (rows * pixelsPerRow) tall.
	fitW := float64(cols)
	fitH := float64(rows) * pixelsPerRow

	scale := min(fitW/imgW, fitH/imgH)
	w := int(imgW * scale)
	h := int(imgH * scale)

	// Round up to even so half-block pairs align.
	if h%2 != 0 {
		h++
	}

	if w < 1 {
		w = 1
	}
	if h < pixelsPerRow {
		h = pixelsPerRow
	}

	resized := resizeImage(img, w, h)

	padLeft := (cols - w) / 2
	pad := strings.Repeat(" ", padLeft)

	var sb strings.Builder
	for y := 0; y < h; y += pixelsPerRow {
		sb.WriteString(pad)
		for x := 0; x < w; x++ {
			tr, tg, tb, _ := resized.At(x, y).RGBA()
			br, bg, bb, _ := resized.At(x, y+1).RGBA()
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				tr>>8, tg>>8, tb>>8, br>>8, bg>>8, bb>>8)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return sb.String()
}

// renderFullscreen renders an image at high quality using the best available
// terminal graphics protocol (kitty or sixel). It scales the image to fill the
// given terminal dimensions while preserving aspect ratio.
func renderFullscreen(img image.Image, termCols, termRows int, gfx graphics) string {
	pxWidth := termCols * cellPixelWidth
	pxHeight := termRows * cellPixelHeight

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()
	scale := min(float64(pxWidth)/float64(imgW), float64(pxHeight)/float64(imgH))
	resized := resizeImage(img, int(float64(imgW)*scale), int(float64(imgH)*scale))

	var buf bytes.Buffer

	if gfx.kitty {
		if err := kittyimg.Fprint(&buf, resized); err != nil {
			slog.Debug("kitty encode failed, falling back", "err", err)
		} else {
			return buf.String()
		}
	}

	if gfx.sixel {
		buf.Reset()
		buf.WriteString("\x1bPq")
		if err := new(sixel.Encoder).Encode(&buf, resized); err != nil {
			slog.Debug("sixel encode failed, falling back", "err", err)
		} else {
			buf.WriteString("\x1b\\")
			return buf.String()
		}
	}

	// No HD graphics protocol available - revert to basic half-block
	// rendering with a notice so the user understands the degradation.
	notice := "  No HD graphics protocol detected - reverted to basic rendering\n" +
		"  (install Kitty, Ghostty, WezTerm, iTerm2, or foot for HD)\n\n"
	return notice + renderPreview(resized, termCols, termRows-3)
}

// resizeImage scales img to the given dimensions using Catmull-Rom
// interpolation.
func resizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}
