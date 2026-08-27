package vt

import "strconv"

func parseParams(s string) []int {
	if s == "" {
		return nil
	}
	var params []int
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ';' {
			part := s[start:i]
			if part == "" {
				params = append(params, 0)
			} else if v, err := strconv.Atoi(part); err == nil {
				params = append(params, v)
			} else {
				params = append(params, 0)
			}
			start = i + 1
		}
	}
	return params
}

func param(p []int, i, def int) int {
	if i >= len(p) || p[i] == 0 {
		return def
	}
	return p[i]
}

func (s *Screen) runCSI(final byte, body string) {
	params := parseParams(body)

	switch final {
	case 'A':
		s.CursorY = clampInt(s.CursorY-param(params, 0, 1), 0, s.Rows-1)
	case 'B':
		s.CursorY = clampInt(s.CursorY+param(params, 0, 1), 0, s.Rows-1)
	case 'C':
		s.CursorX = clampInt(s.CursorX+param(params, 0, 1), 0, s.Cols-1)
	case 'D':
		s.CursorX = clampInt(s.CursorX-param(params, 0, 1), 0, s.Cols-1)
	case 'G':
		s.CursorX = clampInt(param(params, 0, 1)-1, 0, s.Cols-1)
	case 'H', 'f':
		row := param(params, 0, 1)
		col := param(params, 1, 1)
		s.CursorY = clampInt(row-1, 0, s.Rows-1)
		s.CursorX = clampInt(col-1, 0, s.Cols-1)
	case 'J':
		s.eraseDisplay(param(params, 0, 0))
	case 'K':
		s.eraseLine(param(params, 0, 0))
	case 's':
		s.savedX, s.savedY = s.CursorX, s.CursorY
	case 'u':
		s.CursorX, s.CursorY = s.savedX, s.savedY
	case 'm':
		s.applySGR(params)
	}
}

func (s *Screen) eraseDisplay(mode int) {
	switch mode {
	case 0:
		s.eraseLine(0)
		for y := s.CursorY + 1; y < s.Rows; y++ {
			s.Grid[y] = newRow(s.Cols)
		}
	case 1:
		for y := range s.CursorY {
			s.Grid[y] = newRow(s.Cols)
		}
		s.eraseLine(1)
	case 2, 3:
		for y := range s.Rows {
			s.Grid[y] = newRow(s.Cols)
		}
	}
}

func (s *Screen) eraseLine(mode int) {
	row := s.Grid[s.CursorY]
	switch mode {
	case 0:
		for x := s.CursorX; x < s.Cols; x++ {
			row[x] = Cell{Ch: ' ', FG: DefaultColor, BG: DefaultColor}
		}
	case 1:
		for x := 0; x <= s.CursorX && x < s.Cols; x++ {
			row[x] = Cell{Ch: ' ', FG: DefaultColor, BG: DefaultColor}
		}
	case 2:
		s.Grid[s.CursorY] = newRow(s.Cols)
	}
}

func (s *Screen) applySGR(params []int) {
	if len(params) == 0 {
		params = []int{0}
	}
	for _, p := range params {
		switch {
		case p == 0:
			s.curFG, s.curBG, s.bold = DefaultColor, DefaultColor, false
		case p == 1:
			s.bold = true
		case p == 22:
			s.bold = false
		case p == 39:
			s.curFG = DefaultColor
		case p == 49:
			s.curBG = DefaultColor
		case p >= 30 && p <= 37:
			s.curFG = p - 30
		case p >= 90 && p <= 97:
			s.curFG = p - 90 + 8
		case p >= 40 && p <= 47:
			s.curBG = p - 40
		case p >= 100 && p <= 107:
			s.curBG = p - 100 + 8
		}
	}
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
