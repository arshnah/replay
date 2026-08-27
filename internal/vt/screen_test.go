package vt

import "testing"

func gridText(s *Screen) [][]rune {
	out := make([][]rune, s.Rows)
	for y := range s.Rows {
		row := make([]rune, s.Cols)
		for x := range s.Cols {
			row[x] = s.Grid[y][x].Ch
		}
		out[y] = row
	}
	return out
}

func TestAPCSequenceDoesNotLeakIntoGrid(t *testing.T) {
	s := NewScreen(20, 3)
	// kitty's terminal graphics protocol: ESC _ G <base64 payload> ESC \
	// Before the fix, ESC _ fell through to stateNormal and every byte of
	// the payload printed as a literal character.
	s.Write([]byte("before"))
	s.Write([]byte("\x1b_Gf=100,a=T;iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB\x1b\\"))
	s.Write([]byte("after"))

	if s.CursorX != len("beforeafter") || s.CursorY != 0 {
		t.Fatalf("expected only 'beforeafter' to advance the cursor, got (%d,%d)", s.CursorX, s.CursorY)
	}
	got := string(gridText(s)[0])
	want := "beforeafter         "
	if got != want {
		t.Fatalf("APC payload leaked into the grid:\ngot  %q\nwant %q", got, want)
	}
}

func TestDCSSequenceDoesNotLeakIntoGrid(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("hi"))
	s.Write([]byte("\x1bPsome dcs payload\x1b\\"))
	s.Write([]byte("!"))
	got := string(gridText(s)[0])
	if got != "hi!       " {
		t.Fatalf("DCS payload leaked into the grid: %q", got)
	}
}

func TestPlainTextWrapsAndAdvances(t *testing.T) {
	s := NewScreen(5, 3)
	s.Write([]byte("hello"))
	if s.CursorY != 0 || s.CursorX != 5 {
		t.Fatalf("cursor at (%d,%d), want (5,0)", s.CursorX, s.CursorY)
	}
	s.Write([]byte("x"))
	if s.CursorY != 1 || s.CursorX != 1 {
		t.Fatalf("expected wrap to next line, got (%d,%d)", s.CursorX, s.CursorY)
	}
	if s.Grid[1][0].Ch != 'x' {
		t.Fatalf("expected wrapped char on row 1, got %q", s.Grid[1][0].Ch)
	}
}

func TestCarriageReturnAndLineFeed(t *testing.T) {
	s := NewScreen(10, 3)
	s.Write([]byte("abc\r\ndef"))
	if string(s.Grid[0][0].Ch) != "a" || string(s.Grid[1][0].Ch) != "d" {
		t.Fatalf("unexpected grid contents: %q %q", gridText(s)[0], gridText(s)[1])
	}
}

func TestScrollOnOverflow(t *testing.T) {
	s := NewScreen(5, 2)
	s.Write([]byte("row1\r\nrow2\r\nrow3"))
	if s.Grid[0][0].Ch != 'r' || s.Grid[1][0].Ch != 'r' {
		t.Fatalf("unexpected grid after scroll: %+v", gridText(s))
	}
	line0 := string([]rune{s.Grid[0][0].Ch, s.Grid[0][1].Ch, s.Grid[0][2].Ch, s.Grid[0][3].Ch})
	if line0 != "row2" {
		t.Fatalf("expected scrolled content 'row2' on row 0, got %q", line0)
	}
}

func TestCursorMovementCSI(t *testing.T) {
	s := NewScreen(10, 5)
	s.Write([]byte("\x1b[3;5Hx"))
	if s.CursorY != 2 || s.CursorX != 5 {
		t.Fatalf("expected cursor after write at (5,2), got (%d,%d)", s.CursorX, s.CursorY)
	}
	if s.Grid[2][4].Ch != 'x' {
		t.Fatalf("expected x written at row 2 col 4, got %q at that cell", s.Grid[2][4].Ch)
	}
}

func TestSGRColorAndReset(t *testing.T) {
	s := NewScreen(10, 2)
	s.Write([]byte("\x1b[31;1mred\x1b[0mplain"))
	if s.Grid[0][0].FG != 1 || !s.Grid[0][0].Bold {
		t.Fatalf("expected red bold on first cell, got %+v", s.Grid[0][0])
	}
	if s.Grid[0][3].FG != DefaultColor || s.Grid[0][3].Bold {
		t.Fatalf("expected reset color/bold after \\x1b[0m, got %+v", s.Grid[0][3])
	}
}

func TestEraseLine(t *testing.T) {
	s := NewScreen(10, 1)
	s.Write([]byte("abcdefgh\r\x1b[3C\x1b[K"))
	line := string([]rune{s.Grid[0][0].Ch, s.Grid[0][1].Ch, s.Grid[0][2].Ch, s.Grid[0][3].Ch})
	if line != "abc " {
		t.Fatalf("expected erase-to-end after col 3, got %q", gridText(s)[0])
	}
}

func TestOSCSequenceIsIgnoredNotCorrupting(t *testing.T) {
	s := NewScreen(20, 1)
	s.Write([]byte("\x1b]0;window title\x07hello"))
	line := string([]rune{s.Grid[0][0].Ch, s.Grid[0][1].Ch, s.Grid[0][2].Ch, s.Grid[0][3].Ch, s.Grid[0][4].Ch})
	if line != "hello" {
		t.Fatalf("expected OSC to be consumed and 'hello' to print normally, got %q", gridText(s)[0])
	}
}

func TestCloneIsIndependent(t *testing.T) {
	s := NewScreen(5, 1)
	s.Write([]byte("abc"))
	clone := s.Clone()
	s.Write([]byte("de"))
	if clone.Grid[0][3].Ch != ' ' {
		t.Fatalf("clone should not see writes made after cloning, got %q", clone.Grid[0][3].Ch)
	}
	if s.Grid[0][3].Ch != 'd' {
		t.Fatalf("original should reflect the write, got %q", s.Grid[0][3].Ch)
	}
}
