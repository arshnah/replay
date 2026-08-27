package vt

import "unicode/utf8"

type Cell struct {
	Ch   rune
	FG   int
	BG   int
	Bold bool
}

const DefaultColor = -1

type Screen struct {
	Cols, Rows int
	Grid       [][]Cell

	CursorX, CursorY int
	curFG, curBG     int
	bold             bool
	savedX, savedY   int

	state    parseState
	csiBuf   []byte
	oscBuf   []byte
	inEscape byte
	utf8Buf  []byte
}

type parseState int

const (
	stateNormal parseState = iota
	stateEscape
	stateCSI
	stateOSC
)

func NewScreen(cols, rows int) *Screen {
	s := &Screen{
		Cols:  cols,
		Rows:  rows,
		curFG: DefaultColor,
		curBG: DefaultColor,
	}
	s.Grid = make([][]Cell, rows)
	for y := range rows {
		s.Grid[y] = newRow(cols)
	}
	return s
}

func newRow(cols int) []Cell {
	row := make([]Cell, cols)
	for i := range row {
		row[i] = Cell{Ch: ' ', FG: DefaultColor, BG: DefaultColor}
	}
	return row
}

func (s *Screen) Clone() *Screen {
	c := *s
	c.Grid = make([][]Cell, s.Rows)
	for y := range s.Rows {
		row := make([]Cell, s.Cols)
		copy(row, s.Grid[y])
		c.Grid[y] = row
	}
	c.csiBuf = append([]byte(nil), s.csiBuf...)
	c.oscBuf = append([]byte(nil), s.oscBuf...)
	c.utf8Buf = append([]byte(nil), s.utf8Buf...)
	return &c
}

func (s *Screen) Write(data []byte) {
	for _, b := range data {
		s.feed(b)
	}
}

func (s *Screen) feed(b byte) {
	switch s.state {
	case stateNormal:
		s.feedNormal(b)
	case stateEscape:
		s.feedEscape(b)
	case stateCSI:
		s.feedCSI(b)
	case stateOSC:
		s.feedOSC(b)
	}
}

func (s *Screen) feedNormal(b byte) {
	// A byte >= 0x80, or any byte while a multi-byte sequence is already
	// in progress, is UTF-8 continuation/lead data, not a control code or
	// a printable ASCII character on its own. Box-drawing borders, nerd
	// font icons, and anything else outside ASCII are 2-4 bytes each; the
	// old code fed every byte straight to putRune as its own rune, which
	// shredded each real character into that many garbage ones.
	if len(s.utf8Buf) > 0 {
		// A byte that isn't a valid continuation byte (0x80-0xBF) means the
		// sequence in progress was truncated or corrupted. Drop it and
		// reprocess b as if the buffer had been empty, rather than folding
		// an unrelated byte (possibly an ESC) into a doomed decode.
		if b < 0x80 || b > 0xbf {
			s.utf8Buf = s.utf8Buf[:0]
		} else {
			s.utf8Buf = append(s.utf8Buf, b)
			if !utf8.FullRune(s.utf8Buf) {
				return
			}
			r, size := utf8.DecodeRune(s.utf8Buf)
			s.utf8Buf = s.utf8Buf[:0]
			if r != utf8.RuneError || size > 1 {
				s.putRune(r)
			}
			return
		}
	}
	if b >= 0x80 {
		s.utf8Buf = append(s.utf8Buf, b)
		return
	}

	switch b {
	case 0x1b:
		s.state = stateEscape
	case '\r':
		s.CursorX = 0
	case '\n':
		s.lineFeed()
	case '\b':
		if s.CursorX > 0 {
			s.CursorX--
		}
	case '\t':
		next := (s.CursorX/8 + 1) * 8
		if next >= s.Cols {
			next = s.Cols - 1
		}
		s.CursorX = next
	default:
		if b < 0x20 {
			return
		}
		s.putRune(rune(b))
	}
}

func (s *Screen) putRune(r rune) {
	if s.CursorX >= s.Cols {
		s.CursorX = 0
		s.lineFeed()
	}
	s.Grid[s.CursorY][s.CursorX] = Cell{Ch: r, FG: s.curFG, BG: s.curBG, Bold: s.bold}
	s.CursorX++
}

func (s *Screen) lineFeed() {
	if s.CursorY == s.Rows-1 {
		s.scrollUp()
	} else {
		s.CursorY++
	}
}

func (s *Screen) scrollUp() {
	copy(s.Grid, s.Grid[1:])
	s.Grid[s.Rows-1] = newRow(s.Cols)
}

func (s *Screen) feedEscape(b byte) {
	switch b {
	case '[':
		s.state = stateCSI
		s.csiBuf = s.csiBuf[:0]
	case ']', '_', 'P', '^', 'X':
		// OSC (]), APC (_), DCS (P), PM (^) and SOS (X) are all
		// string-terminated the same way (ST = ESC \, some also accept
		// BEL). None of their payloads are rendered, so they all just
		// discard bytes until termination. Without this, an unsupported
		// one (APC is how kitty's terminal graphics protocol embeds a
		// base64 image) falls through to stateNormal below and every byte
		// of its payload gets printed as a literal character.
		s.state = stateOSC
		s.oscBuf = s.oscBuf[:0]
	case '7':
		s.savedX, s.savedY = s.CursorX, s.CursorY
		s.state = stateNormal
	case '8':
		s.CursorX, s.CursorY = s.savedX, s.savedY
		s.state = stateNormal
	default:
		s.state = stateNormal
	}
}

func (s *Screen) feedOSC(b byte) {
	if b == 0x07 || (b == '\\' && len(s.oscBuf) > 0 && s.oscBuf[len(s.oscBuf)-1] == 0x1b) {
		s.state = stateNormal
		return
	}
	s.oscBuf = append(s.oscBuf, b)
}

func (s *Screen) feedCSI(b byte) {
	if b >= 0x40 && b <= 0x7e {
		s.runCSI(b, string(s.csiBuf))
		s.state = stateNormal
		return
	}
	s.csiBuf = append(s.csiBuf, b)
}
