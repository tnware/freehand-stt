// Command brandassets renders Freehand's checked-in platform icon artifacts
// from the canonical SVG artwork under branding. It intentionally supports the
// small, declarative SVG subset used by those sources so generation stays
// reproducible and does not depend on an installed graphics application.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const supersampling = 4

var iconSizes = []int{16, 20, 24, 32, 48, 64, 128, 256}

type document struct {
	minX, minY    float64
	width, height float64
	shapes        []shape
}

type shapeKind uint8

const (
	rectangle shapeKind = iota
	circle
)

type shape struct {
	kind                shapeKind
	x, y, width, height float64
	rx                  float64
	cx, cy, radius      float64
	fill, stroke        *color.NRGBA
	strokeWidth         float64
	role                string
}

type style struct {
	fill        *color.NRGBA
	stroke      *color.NRGBA
	strokeWidth float64
	role        string
}

type artifact struct {
	path string
	data []byte
}

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "verify that generated assets are current")
	flag.Parse()

	outputs, err := generate(*root)
	if err != nil {
		fatal(err)
	}
	if *check {
		if err := checkOutputs(*root, outputs); err != nil {
			fatal(err)
		}
		fmt.Printf("brand assets are current (%d outputs)\n", len(outputs))
		return
	}
	if err := writeOutputs(*root, outputs); err != nil {
		fatal(err)
	}
	fmt.Printf("updated %d generated brand assets\n", len(outputs))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "brand assets:", err)
	os.Exit(1)
}

func generate(root string) ([]artifact, error) {
	appSource, err := os.ReadFile(filepath.Join(root, "branding", "freehand-app-tile.svg"))
	if err != nil {
		return nil, fmt.Errorf("read application icon source: %w", err)
	}
	markSource, err := os.ReadFile(filepath.Join(root, "branding", "freehand-mark.svg"))
	if err != nil {
		return nil, fmt.Errorf("read product mark source: %w", err)
	}
	systemSource, err := os.ReadFile(filepath.Join(root, "branding", "freehand-system-mark.svg"))
	if err != nil {
		return nil, fmt.Errorf("read system mark source: %w", err)
	}

	appDoc, err := parseSVG(appSource)
	if err != nil {
		return nil, fmt.Errorf("parse application icon source: %w", err)
	}
	markDoc, err := parseSVG(markSource)
	if err != nil {
		return nil, fmt.Errorf("parse product mark source: %w", err)
	}
	systemDoc, err := parseSVG(systemSource)
	if err != nil {
		return nil, fmt.Errorf("parse system mark source: %w", err)
	}
	if err := validateSources(appDoc, markDoc, systemDoc); err != nil {
		return nil, err
	}

	appImage := render(appDoc, 1024, nil)
	appMarkDoc := markDoc
	appMarkDoc.minX, appMarkDoc.minY = appDoc.minX, appDoc.minY
	appMarkDoc.width, appMarkDoc.height = appDoc.width, appDoc.height
	draw.Draw(appImage, appImage.Bounds(), render(appMarkDoc, 1024, nil), image.Point{}, draw.Over)
	appIcon, err := encodePNG(appImage)
	if err != nil {
		return nil, fmt.Errorf("encode application icon: %w", err)
	}

	appFrames := make([][]byte, 0, len(iconSizes))
	lightFrames := make([][]byte, 0, len(iconSizes))
	darkFrames := make([][]byte, 0, len(iconSizes))
	lightColour := mustColour("#2563EB")
	darkColour := mustColour("#8AB4FF")
	for _, size := range iconSizes {
		// Every application frame uses the same composition and relative scale.
		// Windows may select a different frame for taskbar and Explorer surfaces,
		// so size-specific artwork would make the application identity jump.
		appImage := render(appDoc, size, nil)
		draw.Draw(appImage, appImage.Bounds(), render(appMarkDoc, size, nil), image.Point{}, draw.Over)
		appFrame, encodeErr := encodePNG(appImage)
		if encodeErr != nil {
			return nil, fmt.Errorf("encode %d-pixel application icon: %w", size, encodeErr)
		}
		lightFrame, encodeErr := encodePNG(render(systemDoc, size, &lightColour))
		if encodeErr != nil {
			return nil, fmt.Errorf("encode %d-pixel light tray icon: %w", size, encodeErr)
		}
		darkFrame, encodeErr := encodePNG(render(systemDoc, size, &darkColour))
		if encodeErr != nil {
			return nil, fmt.Errorf("encode %d-pixel dark tray icon: %w", size, encodeErr)
		}
		appFrames = append(appFrames, appFrame)
		lightFrames = append(lightFrames, lightFrame)
		darkFrames = append(darkFrames, darkFrame)
	}

	return []artifact{
		{path: "build/appicon.png", data: appIcon},
		{path: "build/windows/icon.ico", data: encodeICO(iconSizes, appFrames)},
		{path: "build/windows/tray-light.ico", data: encodeICO(iconSizes, lightFrames)},
		{path: "build/windows/tray-dark.ico", data: encodeICO(iconSizes, darkFrames)},
		{path: "build/appicon.icon/Assets/freehand_mark_vector.svg", data: markSource},
		{path: "frontend/src/lib/assets/freehand-mark.svg", data: markSource},
	}, nil
}

func validateSources(appDoc, markDoc, systemDoc document) error {
	tileCount := 0
	for _, item := range appDoc.shapes {
		if item.role != "tile" {
			return fmt.Errorf("application icon shape has unsupported role %q", item.role)
		}
		tileCount++
	}
	for _, item := range markDoc.shapes {
		if item.role != "glyph" {
			return fmt.Errorf("product mark shape has unsupported role %q", item.role)
		}
	}
	for _, item := range systemDoc.shapes {
		if item.role != "glyph" {
			return fmt.Errorf("system mark shape has unsupported role %q", item.role)
		}
	}
	if tileCount != 1 {
		return fmt.Errorf("application icon contains %d tile shapes, want 1", tileCount)
	}
	return nil
}

func writeOutputs(root string, outputs []artifact) error {
	for _, output := range outputs {
		path := filepath.Join(root, filepath.FromSlash(output.path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create %s parent: %w", output.path, err)
		}
		if err := os.WriteFile(path, output.data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", output.path, err)
		}
	}
	return nil
}

func checkOutputs(root string, outputs []artifact) error {
	var stale []string
	for _, output := range outputs {
		current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(output.path)))
		if err != nil || !bytes.Equal(current, output.data) {
			stale = append(stale, output.path)
		}
	}
	if len(stale) != 0 {
		return fmt.Errorf("generated assets are stale: %s; run `wails3 task common:update:brand-assets`", strings.Join(stale, ", "))
	}
	return nil
}

func parseSVG(data []byte) (document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	doc := document{}
	styles := []style{{fill: colourPtr(mustColour("#000000"))}}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return document{}, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			name := value.Name.Local
			if name == "svg" {
				viewBox, ok := attribute(value.Attr, "viewBox")
				if !ok {
					return document{}, errors.New("svg is missing viewBox")
				}
				parts := strings.Fields(viewBox)
				if len(parts) != 4 {
					return document{}, fmt.Errorf("invalid viewBox %q", viewBox)
				}
				values := make([]float64, 4)
				for index, part := range parts {
					values[index], err = strconv.ParseFloat(part, 64)
					if err != nil {
						return document{}, fmt.Errorf("invalid viewBox %q: %w", viewBox, err)
					}
				}
				doc.minX, doc.minY, doc.width, doc.height = values[0], values[1], values[2], values[3]
				continue
			}
			parent := styles[len(styles)-1]
			current, styleErr := applyStyle(parent, value.Attr)
			if styleErr != nil {
				return document{}, fmt.Errorf("%s style: %w", name, styleErr)
			}
			styles = append(styles, current)
			switch name {
			case "g", "title", "desc":
			case "rect":
				parsed, parseErr := parseRect(value.Attr, current)
				if parseErr != nil {
					return document{}, parseErr
				}
				doc.shapes = append(doc.shapes, parsed)
			case "circle":
				parsed, parseErr := parseCircle(value.Attr, current)
				if parseErr != nil {
					return document{}, parseErr
				}
				doc.shapes = append(doc.shapes, parsed)
			default:
				return document{}, fmt.Errorf("unsupported SVG element %q", name)
			}
		case xml.EndElement:
			if value.Name.Local != "svg" && len(styles) > 1 {
				styles = styles[:len(styles)-1]
			}
		}
	}
	if doc.width <= 0 || doc.height <= 0 || len(doc.shapes) == 0 {
		return document{}, errors.New("svg has no renderable canvas")
	}
	return doc, nil
}

func applyStyle(parent style, attrs []xml.Attr) (style, error) {
	result := parent
	if value, ok := attribute(attrs, "data-role"); ok {
		result.role = value
	}
	if value, ok := attribute(attrs, "fill"); ok {
		parsed, err := parseColour(value)
		if err != nil {
			return style{}, err
		}
		result.fill = parsed
	}
	if value, ok := attribute(attrs, "stroke"); ok {
		parsed, err := parseColour(value)
		if err != nil {
			return style{}, err
		}
		result.stroke = parsed
	}
	if value, ok := attribute(attrs, "stroke-width"); ok {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return style{}, fmt.Errorf("invalid stroke width %q: %w", value, err)
		}
		result.strokeWidth = parsed
	}
	return result, nil
}

func parseRect(attrs []xml.Attr, current style) (shape, error) {
	x, err := numberAttribute(attrs, "x", false)
	if err != nil {
		return shape{}, err
	}
	y, err := numberAttribute(attrs, "y", false)
	if err != nil {
		return shape{}, err
	}
	width, err := numberAttribute(attrs, "width", true)
	if err != nil {
		return shape{}, err
	}
	height, err := numberAttribute(attrs, "height", true)
	if err != nil {
		return shape{}, err
	}
	rx, err := numberAttribute(attrs, "rx", false)
	if err != nil {
		return shape{}, err
	}
	return shape{kind: rectangle, x: x, y: y, width: width, height: height, rx: rx, fill: current.fill, stroke: current.stroke, strokeWidth: current.strokeWidth, role: current.role}, nil
}

func parseCircle(attrs []xml.Attr, current style) (shape, error) {
	cx, err := numberAttribute(attrs, "cx", true)
	if err != nil {
		return shape{}, err
	}
	cy, err := numberAttribute(attrs, "cy", true)
	if err != nil {
		return shape{}, err
	}
	radius, err := numberAttribute(attrs, "r", true)
	if err != nil {
		return shape{}, err
	}
	return shape{kind: circle, cx: cx, cy: cy, radius: radius, fill: current.fill, stroke: current.stroke, strokeWidth: current.strokeWidth, role: current.role}, nil
}

func attribute(attrs []xml.Attr, name string) (string, bool) {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value, true
		}
	}
	return "", false
}

func numberAttribute(attrs []xml.Attr, name string, required bool) (float64, error) {
	value, ok := attribute(attrs, name)
	if !ok {
		if required {
			return 0, fmt.Errorf("missing %s attribute", name)
		}
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s attribute %q: %w", name, value, err)
	}
	return parsed, nil
}

func parseColour(value string) (*color.NRGBA, error) {
	if value == "none" {
		return nil, nil
	}
	if len(value) != 7 || value[0] != '#' {
		return nil, fmt.Errorf("unsupported colour %q", value)
	}
	parsed, err := strconv.ParseUint(value[1:], 16, 24)
	if err != nil {
		return nil, fmt.Errorf("invalid colour %q: %w", value, err)
	}
	result := color.NRGBA{R: uint8(parsed >> 16), G: uint8(parsed >> 8), B: uint8(parsed), A: 0xff}
	return &result, nil
}

func mustColour(value string) color.NRGBA {
	parsed, err := parseColour(value)
	if err != nil || parsed == nil {
		panic("invalid built-in colour: " + value)
	}
	return *parsed
}

func colourPtr(value color.NRGBA) *color.NRGBA { return &value }

func render(doc document, size int, glyphColour *color.NRGBA) *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, size, size))
	sampleCount := supersampling * supersampling
	for py := range size {
		for px := range size {
			var alpha, red, green, blue uint64
			for sy := range supersampling {
				for sx := range supersampling {
					x := doc.minX + (float64(px)+(float64(sx)+0.5)/supersampling)*doc.width/float64(size)
					y := doc.minY + (float64(py)+(float64(sy)+0.5)/supersampling)*doc.height/float64(size)
					pixel := sample(doc.shapes, x, y, glyphColour)
					a := uint64(pixel.A)
					alpha += a
					red += uint64(pixel.R) * a
					green += uint64(pixel.G) * a
					blue += uint64(pixel.B) * a
				}
			}
			if alpha == 0 {
				continue
			}
			result.SetNRGBA(px, py, color.NRGBA{
				R: uint8(red / alpha),
				G: uint8(green / alpha),
				B: uint8(blue / alpha),
				A: uint8(alpha / uint64(sampleCount)),
			})
		}
	}
	return result
}

func sample(shapes []shape, x, y float64, glyphColour *color.NRGBA) color.NRGBA {
	var result color.NRGBA
	for _, item := range shapes {
		paint := item.paintAt(x, y)
		if paint == nil {
			continue
		}
		if glyphColour != nil && item.role == "glyph" {
			result = *glyphColour
		} else {
			result = *paint
		}
	}
	return result
}

func (s shape) paintAt(x, y float64) *color.NRGBA {
	switch s.kind {
	case rectangle:
		if s.stroke != nil && s.strokeWidth > 0 {
			half := s.strokeWidth / 2
			if roundedRectContains(x, y, s.x-half, s.y-half, s.width+s.strokeWidth, s.height+s.strokeWidth, s.rx+half) &&
				!roundedRectContains(x, y, s.x+half, s.y+half, s.width-s.strokeWidth, s.height-s.strokeWidth, math.Max(0, s.rx-half)) {
				return s.stroke
			}
		}
		if s.fill != nil && roundedRectContains(x, y, s.x, s.y, s.width, s.height, s.rx) {
			return s.fill
		}
	case circle:
		distance := math.Hypot(x-s.cx, y-s.cy)
		if s.stroke != nil && s.strokeWidth > 0 && distance <= s.radius+s.strokeWidth/2 && distance >= math.Max(0, s.radius-s.strokeWidth/2) {
			return s.stroke
		}
		if s.fill != nil && distance <= s.radius {
			return s.fill
		}
	}
	return nil
}

func roundedRectContains(px, py, x, y, width, height, radius float64) bool {
	if width <= 0 || height <= 0 || px < x || py < y || px > x+width || py > y+height {
		return false
	}
	radius = math.Min(radius, math.Min(width, height)/2)
	if radius <= 0 || px >= x+radius && px <= x+width-radius || py >= y+radius && py <= y+height-radius {
		return true
	}
	cx := math.Max(x+radius, math.Min(px, x+width-radius))
	cy := math.Max(y+radius, math.Min(py, y+height-radius))
	return math.Hypot(px-cx, py-cy) <= radius
}

func encodePNG(source image.Image) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buffer, source); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func encodeICO(sizes []int, frames [][]byte) []byte {
	const headerSize = 6
	const directoryEntrySize = 16
	result := make([]byte, headerSize+directoryEntrySize*len(frames))
	binary.LittleEndian.PutUint16(result[2:4], 1)
	binary.LittleEndian.PutUint16(result[4:6], uint16(len(frames)))
	offset := len(result)
	for index, frame := range frames {
		entry := result[headerSize+index*directoryEntrySize:]
		size := sizes[index]
		if size < 256 {
			entry[0], entry[1] = byte(size), byte(size)
		}
		binary.LittleEndian.PutUint16(entry[4:6], 1)
		binary.LittleEndian.PutUint16(entry[6:8], 32)
		binary.LittleEndian.PutUint32(entry[8:12], uint32(len(frame)))
		binary.LittleEndian.PutUint32(entry[12:16], uint32(offset))
		result = append(result, frame...)
		offset += len(frame)
	}
	return result
}
