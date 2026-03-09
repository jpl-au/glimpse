package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/charmbracelet/x/ansi/sixel"
	"github.com/dolmen-go/kittyimg"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	// cellPixelWidth is the assumed pixel width of one terminal character
	// cell, used when sizing images for kitty/sixel protocol output.
	cellPixelWidth = 8

	// cellPixelHeight is the assumed pixel height of one terminal character
	// cell.
	cellPixelHeight = 16

	// pixelsPerRow is the number of vertical image pixels encoded per
	// character row when rendering with half-block characters.
	pixelsPerRow = 2
)

// findImageData recursively searches a JSON value for the first string that
// looks like base64-encoded image data. It records matching keys in
// imageFields so that formatValue can show a placeholder without re-decoding.
func findImageData(data any, imageFields map[string]bool) string {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			lk := strings.ToLower(key)
			if strings.Contains(lk, "image") || strings.Contains(lk, "photo") || strings.Contains(lk, "picture") {
				if str, ok := value.(string); ok && isImageString(str) {
					imageFields[key] = true
					return str
				}
			}
			if result := findImageData(value, imageFields); result != "" {
				return result
			}
		}
	case []any:
		for _, item := range v {
			if result := findImageData(item, imageFields); result != "" {
				return result
			}
		}
	case string:
		if isImageString(v) {
			return v
		}
	}
	return ""
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
	drawW := int(imgW * scale)
	drawPxH := int(imgH * scale)
	drawPxH += drawPxH & 1 // round up to even for half-block pairs

	if drawW < 1 {
		drawW = 1
	}
	if drawPxH < pixelsPerRow {
		drawPxH = pixelsPerRow
	}

	resized := resizeImage(img, drawW, drawPxH)

	padLeft := (cols - drawW) / 2
	pad := strings.Repeat(" ", padLeft)

	var sb strings.Builder
	for y := 0; y < drawPxH; y += pixelsPerRow {
		sb.WriteString(pad)
		for x := 0; x < drawW; x++ {
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
func renderFullscreen(img image.Image, termCols, termRows int) string {
	pxWidth := termCols * cellPixelWidth
	pxHeight := termRows * cellPixelHeight

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()
	scale := min(float64(pxWidth)/float64(imgW), float64(pxHeight)/float64(imgH))
	resized := resizeImage(img, int(float64(imgW)*scale), int(float64(imgH)*scale))

	var buf bytes.Buffer

	if supportsKittyGraphics() {
		if err := kittyimg.Fprint(&buf, resized); err == nil {
			return buf.String()
		}
	}

	if supportsSixelGraphics() {
		buf.WriteString("\x1bPq")
		if err := new(sixel.Encoder).Encode(&buf, resized); err == nil {
			buf.WriteString("\x1b\\")
			return buf.String()
		}
	}

	return "No supported image protocol detected.\nRequires: Kitty, Ghostty, WezTerm, iTerm2, or foot"
}

// resizeImage scales img to the given dimensions using Catmull-Rom
// interpolation.
func resizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}
