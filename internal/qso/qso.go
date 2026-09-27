package qso

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var callRe = regexp.MustCompile(`^[A-Z0-9/]+$`)

// QSO is the core log record. Field names map to ADIF 3.1.7 names.
type QSO struct {
	ID        int64
	Call      string // CALL
	QsoDate   string // YYYYMMDD (UTC)
	TimeOn    string // HHMMSS (UTC)
	TimeOff   string // HHMMSS optional
	Band      string // e.g. 20m
	FreqMHz   float64
	Mode      string // e.g. SSB
	Submode   string
	RstSent   string
	RstRcvd   string
	Name      string
	Qth       string
	Comment   string
	TxPower   float64 // watts
	MyCall    string
	MyGrid    string
	MySigInfo string // own POTA park, etc.
	SigInfo   string // contacted park (P2P)
	SotaRef   string
	MySotaRef string
	RawADIF   string // stash for unknown fields on import
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Bands per ADIF Band enumeration.
var Bands = []string{
	"2190m", "630m", "560m", "160m", "80m", "60m", "40m", "30m",
	"20m", "17m", "15m", "12m", "10m", "6m", "4m", "2m",
	"1.25m", "70cm", "33cm", "23cm",
}

type bandEdge struct {
	Name         string
	LowMHz       float64
	HighMHz      float64
}

var bandEdges = []bandEdge{
	{"2190m", 0.136, 0.137},
	{"630m", 0.472, 0.479},
	{"560m", 0.501, 0.504},
	{"160m", 1.8, 2.0},
	{"80m", 3.5, 4.0},
	{"60m", 5.102, 5.405},
	{"40m", 7.0, 7.3},
	{"30m", 10.0, 10.15},
	{"20m", 14.0, 14.35},
	{"17m", 18.068, 18.168},
	{"15m", 21.0, 21.45},
	{"12m", 24.89, 24.99},
	{"10m", 28.0, 29.7},
	{"6m", 50.0, 54.0},
	{"4m", 70.0, 71.0},
	{"2m", 144.0, 148.0},
	{"1.25m", 222.0, 225.0},
	{"70cm", 420.0, 450.0},
	{"33cm", 902.0, 928.0},
	{"23cm", 1240.0, 1300.0},
}

// Modes per ADIF MODE enumeration (common subset, extensible).
var Modes = []string{
	"SSB", "CW", "FM", "AM",
	"FT8", "FT4", "RTTY", "PSK31", "JT65", "JS8",
	"DIGITALVOICE", "C4FM", "DSTAR", "DMR",
}

// BandForFreq derives the ADIF band name from a frequency in MHz.
func BandForFreq(mhz float64) string {
	for _, b := range bandEdges {
		if mhz >= b.LowMHz && mhz <= b.HighMHz {
			return b.Name
		}
	}
	return ""
}

// IsValidBand reports whether band is in the known list (case-insensitive).
func IsValidBand(band string) bool {
	for _, b := range Bands {
		if strings.EqualFold(b, band) {
			return true
		}
	}
	return false
}

// IsValidMode reports whether mode is known (case-insensitive).
func IsValidMode(mode string) bool {
	for _, m := range Modes {
		if strings.EqualFold(m, mode) {
			return true
		}
	}
	return false
}

// NormalizeCall uppercases and trims a callsign.
func NormalizeCall(call string) string {
	return strings.ToUpper(strings.TrimSpace(call))
}

// DefaultRST returns 59 for phone, 599 for CW/digital.
func DefaultRST(mode string) string {
	m := strings.ToUpper(mode)
	switch m {
	case "SSB", "FM", "AM":
		return "59"
	default:
		return "599"
	}
}

// NewQSO returns a QSO prefilled with current UTC date/time and RST defaults.
func NewQSO(myCall, myGrid string) QSO {
	now := time.Now().UTC()
	mode := "SSB"
	rst := DefaultRST(mode)
	return QSO{
		QsoDate:   now.Format("20060102"),
		TimeOn:    now.Format("150405"),
		Band:      "20m",
		Mode:      mode,
		RstSent:   rst,
		RstRcvd:   rst,
		MyCall:    myCall,
		MyGrid:    myGrid,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate checks ADIF minimum: CALL, QSO_DATE, TIME_ON, BAND|FREQ, MODE.
func (q *QSO) Validate() error {
	q.Call = NormalizeCall(q.Call)
	if q.Call == "" {
		return fmt.Errorf("call is required")
	}
	if !callRe.MatchString(q.Call) {
		return fmt.Errorf("invalid callsign %q (A-Z 0-9 / only)", q.Call)
	}
	if _, err := time.Parse("20060102", q.QsoDate); err != nil {
		return fmt.Errorf("invalid QSO_DATE %q (want YYYYMMDD)", q.QsoDate)
	}
	if _, err := time.Parse("150405", q.TimeOn); err != nil {
		if _, err2 := time.Parse("1504", q.TimeOn); err2 != nil {
			return fmt.Errorf("invalid TIME_ON %q (want HHMMSS)", q.TimeOn)
		}
	}
	if q.TimeOff != "" {
		if _, err := time.Parse("150405", q.TimeOff); err != nil {
			if _, err2 := time.Parse("1504", q.TimeOff); err2 != nil {
				return fmt.Errorf("invalid TIME_OFF %q", q.TimeOff)
			}
		}
	}
	if q.Band == "" && q.FreqMHz == 0 {
		return fmt.Errorf("BAND or FREQ is required")
	}
	if q.Band != "" && !IsValidBand(q.Band) {
		return fmt.Errorf("unknown BAND %q", q.Band)
	}
	if q.FreqMHz != 0 && q.Band == "" {
		if b := BandForFreq(q.FreqMHz); b != "" {
			q.Band = b
		}
	}
	if q.Mode == "" {
		return fmt.Errorf("MODE is required")
	}
	q.Mode = strings.ToUpper(q.Mode)
	if !IsValidMode(q.Mode) {
		return fmt.Errorf("unknown MODE %q", q.Mode)
	}
	return nil
}

// DateTimeUTC returns the QSO start as time.Time in UTC.
func (q *QSO) DateTimeUTC() time.Time {
	layouts := []string{"20060102 150405", "20060102 1504"}
	for _, l := range layouts {
		if t, err := time.Parse(l, q.QsoDate+" "+q.TimeOn); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
