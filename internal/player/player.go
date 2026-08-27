package player

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/arshnah/replay/internal/record"
	"github.com/arshnah/replay/internal/vt"
)

type frame struct {
	elapsed time.Duration
	data    []byte
}

type snapshot struct {
	elapsed time.Duration
	frame   int
	screen  *vt.Screen
}

const snapshotEvery = 50

type Session struct {
	Cols, Rows int
	Duration   time.Duration
	frames     []frame
	snapshots  []snapshot
}

func Load(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	magic := make([]byte, len(record.Magic))
	if _, err := readFull(r, magic); err != nil {
		return nil, err
	}
	if string(magic) != record.Magic {
		return nil, fmt.Errorf("not a replay log: bad magic")
	}

	var cols, rows uint32
	if err := binary.Read(r, binary.LittleEndian, &cols); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &rows); err != nil {
		return nil, err
	}

	s := &Session{Cols: int(cols), Rows: int(rows)}

	for {
		var nanos int64
		if err := binary.Read(r, binary.LittleEndian, &nanos); err != nil {
			break
		}
		var length uint32
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			break
		}
		data := make([]byte, length)
		if _, err := readFull(r, data); err != nil {
			break
		}
		s.frames = append(s.frames, frame{elapsed: time.Duration(nanos), data: data})
	}

	s.buildSnapshots()
	if len(s.frames) > 0 {
		s.Duration = s.frames[len(s.frames)-1].elapsed
	}

	return s, nil
}

func readFull(r *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (s *Session) buildSnapshots() {
	screen := vt.NewScreen(s.Cols, s.Rows)
	for i, fr := range s.frames {
		if i%snapshotEvery == 0 {
			s.snapshots = append(s.snapshots, snapshot{
				elapsed: fr.elapsed,
				frame:   i,
				screen:  screen.Clone(),
			})
		}
		screen.Write(fr.data)
	}
}

func (s *Session) ScreenAt(target time.Duration) *vt.Screen {
	if len(s.frames) == 0 {
		return vt.NewScreen(s.Cols, s.Rows)
	}

	idx := 0
	for i, snap := range s.snapshots {
		if snap.elapsed <= target {
			idx = i
		} else {
			break
		}
	}

	var screen *vt.Screen
	startFrame := 0
	if len(s.snapshots) > 0 {
		screen = s.snapshots[idx].screen.Clone()
		startFrame = s.snapshots[idx].frame
	} else {
		screen = vt.NewScreen(s.Cols, s.Rows)
	}

	for i := startFrame; i < len(s.frames); i++ {
		if s.frames[i].elapsed > target {
			break
		}
		screen.Write(s.frames[i].data)
	}

	return screen
}
