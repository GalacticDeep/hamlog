package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

	"hamlog/internal/config"
	"hamlog/internal/qso"
	"hamlog/internal/store"
)

type screen int

const (
	screenLog screen = iota
	screenForm
	screenConfirm
	screenHelp
)

type Model struct {
	db     *store.DB
	cfg    config.Config
	qsos   []qso.QSO
	table  table.Model
	screen screen

	// filtering
	filtering   bool
	filterInput textinput.Model
	filter      string

	// form
	inputs   []textinput.Model
	focusIdx int
	editing  *qso.QSO // nil = new

	// misc
	width   int
	height  int
	status  string
	isError bool
	confirmID int64
	confirmCall string
	showHelp bool

	// path prompt for import
	promptingImport bool
	pathInput textinput.Model
}

type qsosLoadedMsg struct{ qsos []qso.QSO }
type errMsg struct{ err error }
type statusMsg struct {
	text    string
	isError bool
}

func NewModel(db *store.DB, cfg config.Config) Model {
	cols := []table.Column{
		{Title: "Date", Width: 10},
		{Title: "Time", Width: 8},
		{Title: "Call", Width: 12},
		{Title: "Band", Width: 7},
		{Title: "Mode", Width: 8},
		{Title: "RST", Width: 9},
		{Title: "MyPark", Width: 10},
		{Title: "P2P", Width: 10},
		{Title: "Comment", Width: 20},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	s := table.DefaultStyles()
	t.SetStyles(s)

	fi := textinput.New()
	fi.Placeholder = "filter by call..."
	fi.CharLimit = 20

	pi := textinput.New()
	pi.Placeholder = "path to .adi file"
	pi.CharLimit = 256
	pi.Width = 50

	return Model{
		db: db, cfg: cfg,
		table: t,
		filterInput: fi,
		pathInput: pi,
		width: 80, height: 24,
	}
}

func (m *Model) refreshTable() {
	rows := make([]table.Row, 0, len(m.qsos))
	for _, q := range m.qsos {
		rows = append(rows, table.Row{
			q.QsoDate, q.TimeOn, q.Call, q.Band, q.Mode,
			q.RstSent + "/" + q.RstRcvd,
			q.MySigInfo, q.SigInfo, q.Comment,
		})
	}
	m.table.SetRows(rows)
}

func (m *Model) setStatus(text string, isErr bool) {
	m.status = text
	m.isError = isErr
}

// form field definitions in order
func formFields() []string {
	return []string{
		"Call*", "Date YYYYMMDD", "Time HHMMSS", "Band", "Freq MHz",
		"Mode", "RST sent", "RST rcvd", "Name", "QTH",
		"My park", "P2P park", "Comment", "Power W",
	}
}

func (m *Model) buildForm(q *qso.QSO) {
	fields := formFields()
	m.inputs = make([]textinput.Model, len(fields))
	prefill := map[int]string{}
	if q != nil {
		prefill = map[int]string{
			0: q.Call, 1: q.QsoDate, 2: q.TimeOn, 3: q.Band,
			5: q.Mode, 6: q.RstSent, 7: q.RstRcvd, 8: q.Name, 9: q.Qth,
			10: q.MySigInfo, 11: q.SigInfo, 12: q.Comment,
		}
		if q.FreqMHz != 0 {
			prefill[4] = trimFloat(q.FreqMHz)
		}
		if q.TxPower != 0 {
			prefill[13] = trimFloat(q.TxPower)
		}
	} else {
		nq := qso.NewQSO(m.cfg.MyCall, m.cfg.MyGrid)
		prefill = map[int]string{
			1: nq.QsoDate, 2: nq.TimeOn, 3: nq.Band,
			5: nq.Mode, 6: nq.RstSent, 7: nq.RstRcvd,
		}
	}
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f
		ti.CharLimit = 64
		ti.Width = 30
		if v, ok := prefill[i]; ok {
			ti.SetValue(v)
		}
		if i == 0 {
			ti.Focus()
		}
		m.inputs[i] = ti
	}
	m.focusIdx = 0
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
