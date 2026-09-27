# hamlog — amateur radio logger (Go + BubbleTea)

Offline terminal logger for general QSOs, extensible to POTA/SOTA and contesting.
I've been wanting to play around with GO and BubbleTea and this is what I came up with. With help from some open-source LLM models, of course.

## Quick start

```sh
cd ~/hamlog
go build -o hamlog .
./hamlog
```

First run prompts for `My callsign` (+ optional grid). Saved to `~/.config/hamlog/config.json`. DB at `~/.local/share/hamlog/hamlog.db`.

## Keys

- `n` new QSO, `e` edit, `d` delete (confirm `y/n`)
- `/` filter by call, `esc` clear
- `x` export ADIF (`hamlog-YYYYMMDD-HHMMSS.adi` in cwd)
- `i` import ADIF (prompts for path)
- `tab` / `shift+tab` move fields, `enter` save, `esc` cancel
- `ctrl+t` fill UTC date/time in form
- `?` help, `q` / `ctrl+c` quit

## CLI (scripting)

```sh
./hamlog -export log.adi
./hamlog -import log.adi
./hamlog -db /tmp/test.db
```

## ADIF

Minimum fields: `CALL QSO_DATE TIME_ON BAND/FREQ MODE` per ADIF 3.1.7.
POTA: `MY_SIG_INFO` = own park, `SIG_INFO` = P2P park. Aliases `MY_POTA_REF` / `POTA_REF` accepted on import.

Upload exported `.adi` to POTA, QRZ, LoTW, ClubLog.

## Layout

- `main.go` — flags, first-run prompt, TUI launch
- `internal/qso` — validation, bands, modes
- `internal/store` — SQLite CRUD
- `internal/adif` — import/export
- `internal/config` — `~/.config/hamlog`
- `tui/` — BubbleTea model/update/view

Offline v1: no QRZ lookup, spots, rig control, or network.
