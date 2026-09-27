package adif

import (
	"strings"
	"testing"

	"hamlog/internal/qso"
)

func TestExportImportRoundTrip(t *testing.T) {
	qs := []qso.QSO{
		{Call: "W1AW", QsoDate: "20240927", TimeOn: "143000", Band: "20m", FreqMHz: 14.074, Mode: "FT8", RstSent: "599", RstRcvd: "599", MyCall: "N0CALL", MySigInfo: "K-1234", SigInfo: "K-5678", Comment: "P2P"},
	}
	doc := Export(qs, "hamlog", "0.1.0")
	if !strings.Contains(doc, "<CALL:4>W1AW") {
		t.Fatalf("export missing CALL:\n%s", doc)
	}
	if !strings.Contains(doc, "<EOR>") || !strings.Contains(doc, "<EOH>") {
		t.Fatalf("export missing EOH/EOR:\n%s", doc)
	}
	res := Import(doc)
	if res.Skipped != 0 || len(res.QSOs) != 1 {
		t.Fatalf("want 1 imported, got %+v errs=%v", len(res.QSOs), res.Errors)
	}
	got := res.QSOs[0]
	if got.Call != "W1AW" || got.Band != "20m" || got.Mode != "FT8" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if got.MySigInfo != "K-1234" || got.SigInfo != "K-5678" {
		t.Fatalf("POTA refs lost: %+v", got)
	}
}

func TestImportSkipsBadRecords(t *testing.T) {
	doc := "<EOH>\n<CALL:4>W1AW<QSO_DATE:8>20240927<TIME_ON:6>143000<BAND:3>20m<MODE:2>CW<EOR>\n<CALL:3>FOO<EOR>\n"
	res := Import(doc)
	if len(res.QSOs) != 1 || res.Skipped != 1 {
		t.Fatalf("want 1 ok 1 skipped, got %d ok %d skip errs=%v", len(res.QSOs), res.Skipped, res.Errors)
	}
}
