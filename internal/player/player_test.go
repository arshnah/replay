package player

import (
	"bufio"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/arshnah/replay/internal/record"
)

func writeTestLog(t *testing.T, path string, cols, rows int, frames []frame) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	w.WriteString(record.Magic)
	binary.Write(w, binary.LittleEndian, uint32(cols))
	binary.Write(w, binary.LittleEndian, uint32(rows))
	for _, fr := range frames {
		binary.Write(w, binary.LittleEndian, fr.elapsed.Nanoseconds())
		binary.Write(w, binary.LittleEndian, uint32(len(fr.data)))
		w.Write(fr.data)
	}
	w.Flush()
}

func TestScreenAtReconstructsCorrectState(t *testing.T) {
	path := t.TempDir() + "/session.replay"
	frames := []frame{
		{elapsed: 0, data: []byte("a")},
		{elapsed: 10 * time.Millisecond, data: []byte("b")},
		{elapsed: 20 * time.Millisecond, data: []byte("c")},
		{elapsed: 30 * time.Millisecond, data: []byte("d")},
	}
	writeTestLog(t, path, 10, 2, frames)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	screen := s.ScreenAt(20 * time.Millisecond)
	got := string([]rune{screen.Grid[0][0].Ch, screen.Grid[0][1].Ch, screen.Grid[0][2].Ch})
	if got != "abc" {
		t.Fatalf("expected 'abc' at t=20ms, got %q", got)
	}

	screen = s.ScreenAt(5 * time.Millisecond)
	got = string([]rune{screen.Grid[0][0].Ch, screen.Grid[0][1].Ch})
	if got != "a " {
		t.Fatalf("expected 'a ' at t=5ms, got %q", got)
	}
}

func TestScreenAtWithManyFramesUsesSnapshots(t *testing.T) {
	path := t.TempDir() + "/session.replay"
	var frames []frame
	for i := range 500 {
		frames = append(frames, frame{
			elapsed: time.Duration(i) * time.Millisecond,
			data:    []byte{'a' + byte(i%26)},
		})
	}
	writeTestLog(t, path, 100, 5, frames)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.snapshots) < 2 {
		t.Fatalf("expected multiple snapshots for 500 frames, got %d", len(s.snapshots))
	}

	end := s.ScreenAt(s.Duration)
	mid := s.ScreenAt(s.Duration / 2)
	if end == mid {
		t.Fatalf("expected distinct screen objects")
	}
}

func TestScreenAtEmptySessionDoesNotPanic(t *testing.T) {
	path := t.TempDir() + "/empty.replay"
	writeTestLog(t, path, 10, 2, nil)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	screen := s.ScreenAt(0)
	if screen.Cols != 10 || screen.Rows != 2 {
		t.Fatalf("unexpected empty screen dims: %dx%d", screen.Cols, screen.Rows)
	}
}
