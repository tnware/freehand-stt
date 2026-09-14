package managedruntime

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	displayText byte = iota
	displayEscape
	displayEscapeIntermediate
	displayCSI
	displayDiscardCSI
	displayOSC
	displayString
	displayOSCEscape
	displayStringEscape
)

// processDisplayDecoder is ONLY the opted-in viewer's display policy, never the
// metadata/diagnostic parser. No partial escape reaches the renderer. Memory for
// incomplete UTF-8/CSI is fixed; string-control payloads are discarded in place.
// Allowed: CR/BS/HT/LF, validated semicolon SGR, and CSI K with mode 0/1/2.
// No cursor/window/mode controls, OSC, DCS, APC, PM, SOS or C1 passthrough.
// Complete tokens are independent of stream interleaving and chunk eviction;
// a truncated tail may lose earlier styling, but cannot inherit parser state.
type processDisplayDecoder struct {
	state    byte
	params   [64]byte
	nparams  int
	pending  [utf8.UTFMax]byte
	npending int
}

func (d *processDisplayDecoder) write(p []byte) string {
	if d.npending > 0 {
		p = append(append([]byte(nil), d.pending[:d.npending]...), p...)
		d.npending = 0
	}
	var out strings.Builder
	for len(p) > 0 {
		if !utf8.FullRune(p) {
			d.npending = copy(d.pending[:], p)
			break
		}
		r, n := utf8.DecodeRune(p)
		p = p[n:]
		d.rune(&out, r)
	}
	return out.String()
}

func (d *processDisplayDecoder) rune(out *strings.Builder, r rune) {
	// BEL terminates OSC only. DCS/APC/PM/SOS require ST. Embedded escapes
	// cannot turn a hostile string payload into a permitted display sequence.
	switch d.state {
	case displayOSC, displayString, displayOSCEscape, displayStringEscape:
		osc := d.state == displayOSC || d.state == displayOSCEscape
		escaped := d.state == displayOSCEscape || d.state == displayStringEscape
		if r == 0x9c || escaped && r == '\\' || osc && r == '\a' {
			d.state = displayText
		} else if r == '\x1b' {
			if osc {
				d.state = displayOSCEscape
			} else {
				d.state = displayStringEscape
			}
		} else if osc {
			d.state = displayOSC
		} else {
			d.state = displayString
		}
		return
	}
	// ESC restarts an incomplete CSI; CAN/SUB cancel it without emission.
	switch r {
	case '\x1b':
		d.state, d.nparams = displayEscape, 0
		return
	case 0x18, 0x1a:
		d.state, d.nparams = displayText, 0
		return
	case 0x9b:
		d.state, d.nparams = displayDiscardCSI, 0
		return
	case 0x9d:
		d.state = displayOSC
		return
	case 0x90, 0x98, 0x9e, 0x9f:
		d.state = displayString
		return
	}
	switch d.state {
	case displayEscape:
		switch r {
		case '[':
			d.state = displayCSI
		case ']':
			d.state = displayOSC
		case 'P', 'X', '^', '_':
			d.state = displayString
		default:
			if r >= 0x20 && r <= 0x2f {
				d.state = displayEscapeIntermediate
			} else {
				d.state = displayText
			}
		}
		return
	case displayEscapeIntermediate:
		if r < 0x20 || r > 0x2f {
			d.state = displayText
		}
		return
	case displayCSI, displayDiscardCSI:
		if r >= 0x40 && r <= 0x7e {
			params := string(d.params[:d.nparams])
			if d.state == displayCSI && allowedProcessDisplayCSI(params, r) {
				out.WriteString("\x1b[")
				out.WriteString(params)
				out.WriteRune(r)
			}
			d.state, d.nparams = displayText, 0
		} else if d.state == displayCSI {
			if (r >= '0' && r <= '9' || r == ';') && d.nparams < len(d.params) {
				d.params[d.nparams] = byte(r)
				d.nparams++
			} else {
				d.state, d.nparams = displayDiscardCSI, 0
			}
		}
		return
	}
	if r == '\r' || r == '\b' || r == '\t' || r == '\n' || !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r) {
		out.WriteRune(r)
	}
}

func allowedProcessDisplayCSI(params string, final rune) bool {
	if final == 'K' {
		return params == "" || params == "0" || params == "1" || params == "2"
	}
	if final != 'm' {
		return false
	}
	if params == "" {
		return true
	} // SGR default reset.
	// No colon/subparameters, private markers, intermediates or empty fields.
	// At most 16 decimal parameters, each 1-3 digits and in [0,255].
	var values [16]int
	n := 0
	for _, field := range strings.Split(params, ";") {
		if n == len(values) || len(field) == 0 || len(field) > 3 {
			return false
		}
		v := 0
		for _, c := range field {
			if c < '0' || c > '9' {
				return false
			}
			v = v*10 + int(c-'0')
		}
		if v > 255 {
			return false
		}
		values[n] = v
		n++
	}
	for i := 0; i < n; i++ {
		v := values[i]
		switch {
		case v == 38 || v == 48:
			// Indexed (5;n) or RGB (2;r;g;b) foreground/background only.
			if i+2 < n && values[i+1] == 5 {
				i += 2
			} else if i+4 < n && values[i+1] == 2 {
				i += 4
			} else {
				return false
			}
		case v >= 0 && v <= 5, v >= 7 && v <= 9,
			v >= 21 && v <= 25, v >= 27 && v <= 29,
			v >= 30 && v <= 37, v == 39, v >= 40 && v <= 47, v == 49,
			v == 53, v == 55, v >= 90 && v <= 97, v >= 100 && v <= 107:
		default:
			return false
		}
	}
	return true
}

// Input is decoder output, so every ESC starts a complete, bounded CSI m/K.
// Keep that whole token (and every UTF-8 rune) on one side of the chunk cut.
func processDisplayChunkSize(text string, limit int) int {
	end := 0
	for end < len(text) {
		_, n := utf8.DecodeRuneInString(text[end:])
		if text[end] == '\x1b' {
			n = strings.IndexAny(text[end:], "mK") + 1
		}
		if end+n > limit {
			break
		}
		end += n
	}
	return end
}
