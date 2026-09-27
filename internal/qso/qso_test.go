package qso

import (
	"strings"
	"testing"
)

func TestValidateMinimum(t *testing.T) {
	q := QSO{Call: "dl1abc", QsoDate: "20240927", TimeOn: "143000", Band: "20m", Mode: "SSB"}
	if err := q.Validate(); err != nil {
		t.Fatalf("want valid, got %v", err)
	}
	if q.Call != "DL1ABC" {
		t.Fatalf("want uppercased call, got %q", q.Call)
	}
}

func TestValidateRequiresBandOrFreq(t *testing.T) {
	q := QSO{Call: "W1AW", QsoDate: "20240927", TimeOn: "143000", Mode: "CW"}
	if err := q.Validate(); err == nil {
		t.Fatal("want error for missing band/freq")
	}
}

func TestBandForFreq(t *testing.T) {
	if got := BandForFreq(14.074); got != "20m" {
		t.Fatalf("got %q want 20m", got)
	}
	if got := BandForFreq(7.074); got != "40m" {
		t.Fatalf("got %q want 40m", got)
	}
	if got := BandForFreq(9999); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestDefaultRST(t *testing.T) {
	if DefaultRST("SSB") != "59" {
		t.Fatal("SSB should default 59")
	}
	if DefaultRST("FT8") != "599" {
		t.Fatal("FT8 should default 599")
	}
	if strings.ToUpper("cw") != "CW" {
		t.Fatal("sanity")
	}
}
