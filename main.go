package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hamlog/internal/adif"
	"hamlog/internal/config"
	"hamlog/internal/store"
	"hamlog/tui"
)

func main() {
	var (
		exportPath = flag.String("export", "", "export all QSOs to ADIF file and exit")
		importPath = flag.String("import", "", "import ADIF file and exit")
		dbPath     = flag.String("db", "", "override sqlite path")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	if *dbPath != "" {
		cfg.DBPath = *dbPath
	}

	// First-run prompt for operator callsign (approved behavior).
	if strings.TrimSpace(cfg.MyCall) == "" && *exportPath == "" && *importPath == "" {
		cfg.MyCall = promptCall()
		// grid optional
		fmt.Print("My grid square (optional, e.g. FN20): ")
		grid, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		cfg.MyGrid = strings.ToUpper(strings.TrimSpace(grid))
		if err := cfg.Save(); err != nil {
			fmt.Fprintln(os.Stderr, "save config:", err)
		}
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer db.Close()

	if *exportPath != "" {
		qsos, err := db.List("", 100000)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		doc := adif.Export(qsos, "hamlog", "0.1.0")
		if err := os.WriteFile(*exportPath, []byte(doc), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("Exported %d QSOs → %s\n", len(qsos), *exportPath)
		return
	}

	if *importPath != "" {
		data, err := os.ReadFile(*importPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		res := adif.Import(string(data))
		inserted := 0
		for _, q := range res.QSOs {
			qq := q
			if qq.MyCall == "" {
				qq.MyCall = cfg.MyCall
			}
			if _, err := db.Insert(&qq); err != nil {
				res.Skipped++
				continue
			}
			inserted++
		}
		fmt.Printf("Imported %d, skipped %d from %s\n", inserted, res.Skipped, *importPath)
		for _, e := range res.Errors {
			fmt.Fprintln(os.Stderr, " -", e)
		}
		return
	}

	m := tui.NewModel(db, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func promptCall() string {
	fmt.Print("My callsign (e.g. N0CALL): ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.ToUpper(strings.TrimSpace(line))
}
