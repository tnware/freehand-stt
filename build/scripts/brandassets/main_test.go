package main

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestParseAndRenderSupportedSVG(t *testing.T) {
	doc, err := parseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect data-role="tile" x="2" y="2" width="60" height="60" rx="12" fill="#FAFAFC" stroke="#D6DCE8" stroke-width="2"/><g data-role="glyph" fill="#2563EB"><circle cx="32" cy="32" r="8"/></g></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.shapes) != 2 || doc.shapes[0].role != "tile" || doc.shapes[1].role != "glyph" {
		t.Fatalf("parsed shapes = %#v", doc.shapes)
	}
	image := render(doc, 16, nil)
	if image.NRGBAAt(8, 8).A == 0 || image.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("rendered alpha center=%d corner=%d", image.NRGBAAt(8, 8).A, image.NRGBAAt(0, 0).A)
	}
}

func TestEncodeICOContainsEveryDeclaredSize(t *testing.T) {
	frames := make([][]byte, 0, len(iconSizes))
	for _, size := range iconSizes {
		encoded, err := encodePNG(render(document{
			width: 1, height: 1,
			shapes: []shape{{kind: rectangle, width: 1, height: 1, fill: colourPtr(mustColour("#2563EB"))}},
		}, size, nil))
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, encoded)
	}

	encoded := encodeICO(iconSizes, frames)
	if got := int(binary.LittleEndian.Uint16(encoded[4:6])); got != len(iconSizes) {
		t.Fatalf("ICO image count = %d, want %d", got, len(iconSizes))
	}
	for index, want := range iconSizes {
		entry := encoded[6+index*16:]
		got := int(entry[0])
		if got == 0 {
			got = 256
		}
		if got != want {
			t.Errorf("ICO entry %d size = %d, want %d", index, got, want)
		}
		offset := int(binary.LittleEndian.Uint32(entry[12:16]))
		length := int(binary.LittleEndian.Uint32(entry[8:12]))
		decoded, err := png.Decode(bytes.NewReader(encoded[offset : offset+length]))
		if err != nil {
			t.Fatalf("decode ICO entry %d: %v", index, err)
		}
		if decoded.Bounds().Dx() != want || decoded.Bounds().Dy() != want {
			t.Errorf("ICO entry %d dimensions = %v, want %dx%d", index, decoded.Bounds(), want, want)
		}
	}
}

func TestParseSVGRejectsUnsupportedContent(t *testing.T) {
	_, err := parseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><path d="M0 0"/></svg>`))
	if err == nil {
		t.Fatal("unsupported path was accepted")
	}
}

func TestValidateSourcesRejectsGlyphInApplicationTile(t *testing.T) {
	mark := document{width: 1, height: 1, shapes: []shape{{kind: circle, radius: 1, role: "glyph"}}}
	app := document{width: 1, height: 1, shapes: []shape{
		{kind: rectangle, width: 1, height: 1, role: "tile"},
		{kind: circle, radius: 0.5, role: "glyph"},
	}}
	system := document{width: 1, height: 1, shapes: []shape{{kind: rectangle, width: 1, height: 1, role: "glyph"}}}
	if err := validateSources(app, mark, system); err == nil {
		t.Fatal("duplicated application glyph was accepted")
	}
}
