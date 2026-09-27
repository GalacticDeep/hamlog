package tui

import (
	"testing"

	"hamlog/internal/config"
)

func TestResetForNextQSO(t *testing.T) {
	m := NewModel(nil, config.Config{})
	m.buildForm(nil)
	// simulate filled form
	m.inputs[0].SetValue("W1AW")
	m.inputs[3].SetValue("20m")
	m.inputs[4].SetValue("14.074")
	m.inputs[5].SetValue("FT8")
	m.inputs[6].SetValue("599")
	m.inputs[8].SetValue(" somebody")
	m.inputs[10].SetValue("K-1234")
	m.inputs[11].SetValue("K-5678")
	m.inputs[12].SetValue("hello")
	m.inputs[13].SetValue("100")

	m.resetForNextQSO("20m", "14.074", "FT8", "100", "K-1234")

	if got := m.inputs[0].Value(); got != "" {
		t.Fatalf("call should clear, got %q", got)
	}
	if got := m.inputs[3].Value(); got != "20m" {
		t.Fatalf("band sticky, got %q", got)
	}
	if got := m.inputs[4].Value(); got != "14.074" {
		t.Fatalf("freq sticky, got %q", got)
	}
	if got := m.inputs[5].Value(); got != "FT8" {
		t.Fatalf("mode sticky, got %q", got)
	}
	if got := m.inputs[13].Value(); got != "100" {
		t.Fatalf("power sticky, got %q", got)
	}
	if got := m.inputs[10].Value(); got != "K-1234" {
		t.Fatalf("my park sticky, got %q", got)
	}
	if got := m.inputs[11].Value(); got != "" {
		t.Fatalf("P2P park should clear, got %q", got)
	}
	if got := m.inputs[12].Value(); got != "" {
		t.Fatalf("comment should clear, got %q", got)
	}
	if got := m.inputs[1].Value(); len(got) != 8 {
		t.Fatalf("date should refresh YYYYMMDD, got %q", got)
	}
	if got := m.inputs[2].Value(); len(got) != 6 {
		t.Fatalf("time should refresh HHMMSS, got %q", got)
	}
	if m.focusIdx != 0 {
		t.Fatalf("focus should return to call, got %d", m.focusIdx)
	}
}
