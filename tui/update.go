package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"hamlog/internal/adif"
	"hamlog/internal/qso"
)

func (m Model) Init() tea.Cmd {
	return m.loadQSOs
}

func (m Model) loadQSOs() tea.Msg {
	qsos, err := m.db.List(m.filter, 2000)
	if err != nil {
		return errMsg{err}
	}
	return qsosLoadedMsg{qsos}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(max(5, msg.Height-10))
		return m, nil

	case qsosLoadedMsg:
		m.qsos = msg.qsos
		m.refreshTable()
		return m, nil

	case errMsg:
		m.setStatus("Error: "+msg.err.Error(), true)
		return m, nil

	case statusMsg:
		m.setStatus(msg.text, msg.isError)
		return m, m.loadQSOs

	case formSavedMsg:
		m.setStatus(msg.text, false)
		if msg.isEdit {
			m.screen = screenLog
			m.editing = nil
			return m, m.loadQSOs
		}
		// Rapid logging: stay in the form, keep sticky fields,
		// refresh date/time.
		m.resetForNextQSO(msg.band, msg.freq, msg.mode, msg.power, msg.myPark)
		return m, m.loadQSOs
	}

	// import path prompt takes over
	if m.promptingImport {
		return m.updateImportPrompt(msg)
	}

	switch m.screen {
	case screenForm:
		return m.updateForm(msg)
	case screenConfirm:
		return m.updateConfirm(msg)
	default:
		return m.updateLog(msg)
	}
}

func (m Model) updateLog(msg tea.Msg) (tea.Model, tea.Cmd) {
	// filter input active
	if m.filtering {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.filter = strings.ToUpper(strings.TrimSpace(m.filterInput.Value()))
				m.filtering = false
				m.filterInput.Blur()
				m.setStatus("Filter: "+m.filter, false)
				return m, m.loadQSOs
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filter = ""
				return m, m.loadQSOs
			}
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n":
			m.editing = nil
			m.buildForm(nil)
			m.screen = screenForm
			return m, nil
		case "e":
			q := m.selectedQSO()
			if q == nil {
				m.setStatus("No QSO selected", true)
				return m, nil
			}
			cp := *q
			m.editing = &cp
			m.buildForm(&cp)
			m.screen = screenForm
			return m, nil
		case "d":
			q := m.selectedQSO()
			if q == nil {
				m.setStatus("No QSO selected", true)
				return m, nil
			}
			m.confirmID = q.ID
			m.confirmCall = q.Call
			m.screen = screenConfirm
			return m, nil
		case "/":
			m.filtering = true
			m.filterInput.Focus()
			return m, nil
		case "x":
			return m, m.exportCmd()
		case "i":
			m.promptingImport = true
			m.pathInput.Focus()
			m.pathInput.SetValue("")
			return m, nil
		case "?", "h":
			m.screen = screenHelp
			return m, nil
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.filterInput.SetValue("")
				return m, m.loadQSOs
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) selectedQSO() *qso.QSO {
	if len(m.qsos) == 0 {
		return nil
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.qsos) {
		return nil
	}
	return &m.qsos[idx]
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		// forward to focused input
		var cmd tea.Cmd
		m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		return m, cmd
	}
	switch key.String() {
	case "esc":
		m.screen = screenLog
		m.setStatus("Cancelled", false)
		return m, nil
	case "enter":
		return m, m.saveForm()
	case "tab":
		m.focusIdx = (m.focusIdx + 1) % len(m.inputs)
		for i := range m.inputs {
			if i == m.focusIdx {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		return m, nil
	case "shift+tab":
		m.focusIdx = (m.focusIdx - 1 + len(m.inputs)) % len(m.inputs)
		for i := range m.inputs {
			if i == m.focusIdx {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		return m, nil
	}
	// "t" fills UTC now when in date/time fields
	if key.String() == "t" && (m.focusIdx == 1 || m.focusIdx == 2) && m.inputs[m.focusIdx].Value() == "" {
		// let it fall through to input; handle ctrl+t instead below
	}
	if key.String() == "ctrl+t" {
		now := time.Now().UTC()
		m.inputs[1].SetValue(now.Format("20060102"))
		m.inputs[2].SetValue(now.Format("150405"))
		return m, nil
	}
	// single "t" with empty date/time also fills (convenience, only when field empty and no modifiers)
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	// after typing, if user pressed "t" as first char in date/time field... keep simple: skip magic
	_ = cmd
	return m, cmd
}

func (m Model) saveForm() tea.Cmd {
	return func() tea.Msg {
		get := func(i int) string { return strings.TrimSpace(m.inputs[i].Value()) }
		var freq, pwr float64
		if v := get(4); v != "" {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return errMsg{fmt.Errorf("invalid freq %q", v)}
			}
			freq = f
		}
		if v := get(13); v != "" {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return errMsg{fmt.Errorf("invalid power %q", v)}
			}
			pwr = f
		}
		if m.editing != nil {
			q := *m.editing
			q.Call = get(0)
			q.QsoDate = get(1)
			q.TimeOn = get(2)
			q.Band = strings.ToLower(get(3))
			q.FreqMHz = freq
			q.Mode = get(5)
			q.RstSent = get(6)
			q.RstRcvd = get(7)
			q.Name = get(8)
			q.Qth = get(9)
			q.MySigInfo = get(10)
			q.SigInfo = get(11)
			q.Comment = get(12)
			q.TxPower = pwr
			if q.MyCall == "" {
				q.MyCall = m.cfg.MyCall
			}
			if q.MyGrid == "" {
				q.MyGrid = m.cfg.MyGrid
			}
			if err := m.db.Update(&q); err != nil {
				return errMsg{err}
			}
			return formSavedMsg{text: "Updated " + q.Call, isEdit: true, call: q.Call}
		}
		band := strings.ToLower(get(3))
		freqStr := get(4)
		mode := get(5)
		myPark := get(10)
		powerStr := get(13)
		q := qso.QSO{
			Call: get(0), QsoDate: get(1), TimeOn: get(2),
			Band: band, FreqMHz: freq,
			Mode: mode, RstSent: get(6), RstRcvd: get(7),
			Name: get(8), Qth: get(9), MySigInfo: get(10), SigInfo: get(11),
			Comment: get(12), TxPower: pwr,
			MyCall: m.cfg.MyCall, MyGrid: m.cfg.MyGrid,
		}
		if _, err := m.db.Insert(&q); err != nil {
			return errMsg{err}
		}
		return formSavedMsg{
			text: "Logged " + q.Call, isEdit: false, call: q.Call,
			band: band, freq: freqStr, mode: mode, power: powerStr, myPark: myPark,
		}
	}
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch strings.ToLower(msg.String()) {
		case "y", "enter":
			id := m.confirmID
			m.screen = screenLog
			return m, func() tea.Msg {
				if err := m.db.Delete(id); err != nil {
					return errMsg{err}
				}
				return statusMsg{"Deleted", false}
			}
		case "n", "esc", "q":
			m.screen = screenLog
			m.setStatus("Cancelled", false)
			return m, nil
		}
	}
	return m, nil
}

func (m Model) exportCmd() tea.Cmd {
	return func() tea.Msg {
		qsos, err := m.db.List("", 100000)
		if err != nil {
			return errMsg{err}
		}
		doc := adif.Export(qsos, "hamlog", "0.1.0")
		name := "hamlog-" + time.Now().UTC().Format("20060102-150405") + ".adi"
		if err := os.WriteFile(name, []byte(doc), 0o644); err != nil {
			return errMsg{err}
		}
		return statusMsg{fmt.Sprintf("Exported %d QSOs → %s", len(qsos), name), false}
	}
}

func (m Model) updateImportPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.promptingImport = false
			m.pathInput.Blur()
			return m, nil
		case "enter":
			path := strings.TrimSpace(m.pathInput.Value())
			m.promptingImport = false
			m.pathInput.Blur()
			if path == "" {
				return m, nil
			}
			return m, m.importCmd(path)
		}
	}
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m Model) importCmd(path string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(path)
		if err != nil {
			return errMsg{err}
		}
		res := adif.Import(string(data))
		inserted := 0
		for _, q := range res.QSOs {
			qq := q
			if qq.MyCall == "" {
				qq.MyCall = m.cfg.MyCall
			}
			if qq.MyGrid == "" {
				qq.MyGrid = m.cfg.MyGrid
			}
			if _, err := m.db.Insert(&qq); err != nil {
				res.Skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", qq.Call, err))
				continue
			}
			inserted++
		}
		msg := fmt.Sprintf("Imported %d, skipped %d from %s", inserted, res.Skipped, path)
		if len(res.Errors) > 0 && inserted == 0 {
			return errMsg{fmt.Errorf("%s (%s)", msg, strings.Join(res.Errors, "; "))}
		}
		// reload table after import
		qsos, err := m.db.List(m.filter, 2000)
		if err != nil {
			return errMsg{err}
		}
		_ = qsos
		return statusMsg{msg, false}
	}
}

// table.Model Update returns table.Model; keep type assert helper
var _ = table.New

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
