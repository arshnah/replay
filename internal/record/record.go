package record

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

const Magic = "REPLAY01"

func Record(logPath string, cols, rows int, args []string) error {
	cmd := exec.Command(args[0], args[1:]...)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}
	defer ptmx.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		for range sigCh {
			pty.InheritSize(os.Stdin, ptmx)
		}
	}()

	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	w := bufio.NewWriter(logFile)
	defer w.Flush()

	if err := writeHeader(w, cols, rows); err != nil {
		return err
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	go io.Copy(ptmx, os.Stdin)

	start := time.Now()
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			os.Stdout.Write(buf[:n])
			writeFrame(w, time.Since(start), buf[:n])
		}
		if err != nil {
			break
		}
	}

	cmd.Wait()
	return nil
}

func writeHeader(w *bufio.Writer, cols, rows int) error {
	if _, err := w.WriteString(Magic); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(cols)); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, uint32(rows))
}

func writeFrame(w *bufio.Writer, elapsed time.Duration, data []byte) {
	binary.Write(w, binary.LittleEndian, elapsed.Nanoseconds())
	binary.Write(w, binary.LittleEndian, uint32(len(data)))
	w.Write(data)
}
