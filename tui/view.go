package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	header := m.viewHeader()
	footer := m.viewFooter()
	var body string
	switch {
	case m.promptingImport:
		body = m.viewImportPrompt()
	case m.screen == screenForm:
		body = m.viewForm()
	case m.screen == screenConfirm:
		body = m.viewConfirm()
	case m.screen == screenHelp:
		body = m.viewHelp()
	default:
		body = m.viewLog()
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) viewHeader() string {
	n := len(m.qsos)
	title := fmt.Sprintf("hamlog  ·  %s  ·  %d QSOs", m.cfg.MyCall, n)
	if m.filter != "" {
		title += "  ·  filter: " + m.filter
	}
	return HeaderStyle.Render(title)
}

func (m Model) viewFooter() string {
	var sb strings.Builder
	if m.isError {
		sb.WriteString(ErrorStyle.Render(m.status))
	} else if m.status != "" {
		sb.WriteString(StatusStyle.Render(m.status))
	}
	sb.WriteString("\n")
	sb.WriteString(HelpStyle.Render(footerKeys))
	return sb.String()
}

func (m Model) viewLog() string {
	if m.filtering {
		return "Filter: " + m.filterInput.View() + "  (enter apply · esc clear)\n\n" + m.table.View()
	}
	if len(m.qsos) == 0 {
		return "No QSOs yet. Press n to log your first contact.\n\n" + m.table.View()
	}
	return m.table.View()
}

func (m Model) viewForm() string {
	var sb strings.Builder
	if m.editing != nil {
		sb.WriteString(SelectedStyle.Render("Edit QSO"))
	} else {
		sb.WriteString(SelectedStyle.Render("New QSO (UTC)"))
	}
	sb.WriteString("\n" + HelpStyle.Render(formHelp+" · ctrl+t fill UTC now") + "\n\n")
	fields := formFields()
	for i, f := range fields {
		cursor := "  "
		if i == m.focusIdx {
			cursor = "▸ "
		}
		sb.WriteString(fmt.Sprintf("%s%-14s %s\n", cursor, f, m.inputs[i].View()))
	}
	return sb.String()
}

func (m Model) viewConfirm() string {
	return BoxStyle.Render(fmt.Sprintf("Delete QSO %s (id %d)?\n\n[y]es / [n]o", m.confirmCall, m.confirmID))
}

func (m Model) viewHelp() string {
	return BoxStyle.Render(`hamlog — amateur radio logger (offline)

  n          new QSO            e  edit selected
  d          delete (confirm)   /  filter by call
  x          export ADIF        i  import ADIF
  enter      save / confirm     esc cancel / clear filter
  tab        next field (form)  ctrl+t fill UTC now (form)
  q / ctrl+c quit

ADIF: CALL QSO_DATE TIME_ON BAND/FREQ MODE (+ POTA MY_SIG_INFO/SIG_INFO).
Export writes hamlog-YYYYMMDD-HHMMSS.adi to current dir.

Press esc or q to go back.`)
}

func (m Model) viewImportPrompt() string {
	return BoxStyle.Render("Import ADIF path:\n\n" + m.pathInput.View() + "\n\nenter import · esc cancel")
}
