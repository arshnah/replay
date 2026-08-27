# replay

Records a terminal session to a file, then lets you scrub back and forth
through it instead of only replaying linearly like `script`/`asciinema` do.

## Install

```
go install github.com/arshnah/replay@latest
```

Or build from source:

```
git clone https://github.com/arshnah/replay
cd replay
go build -o replay .
```

## Usage

Record:

```
replay record session.replay -- bash
```

Runs `bash` (or any command) in a real pty, mirrors its output to your
terminal live, and logs every write with a timestamp to `session.replay`.

Play back:

```
replay play session.replay
```

Opens a scrub UI. Controls:

- `h`/`l` (or left/right): seek by 1% of session length
- `H`/`L`: seek by 10%
- `g`/`G`: jump to start/end
- `space`: toggle auto-play
- `q`: quit

## How scrubbing works

The log is an append-only sequence of `(timestamp, byte length, raw bytes)`
frames. A small ANSI/VT100 terminal emulator (`internal/vt`) interprets those
bytes into a 2D cell grid, character plus foreground/background color plus
bold, the same way a real terminal does. Every 50 frames a full snapshot of
that grid gets cached in memory. Scrubbing to any timestamp finds the
nearest snapshot at or before it and replays only the remaining frames
forward from there, instead of replaying the whole session from frame zero
every time you move the scrub bar.

The VT emulator covers cursor movement, erase-line/erase-display, basic SGR
colors and bold, scrolling, tab stops, and OSC sequences (consumed and
ignored, so window-title-setting programs don't corrupt the parse). It does
not implement the alternate screen buffer, scroll regions, or full unicode
width handling. Scope choices, not oversights. Most CLI output doesn't
touch those.

## License

See [LICENSE](LICENSE).
