package adif

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"hamlog/internal/qso"
)

const Version = "3.1.7"

// fieldRe matches <NAME:LEN[:TYPE]>value
var fieldRe = regexp.MustCompile(`(?i)<([A-Z0-9_]+):(\d+)(?::[A-Z])?>([^<]*)`)

// Export renders QSOs as an .adi document.
func Export(qsos []qso.QSO, programID, programVersion string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("hamlog ADIF export\n<ADIF_VER:%d>%s\n<PROGRAMID:%d>%s\n<PROGRAMVERSION:%d>%s\n<EOH>\n",
		len(Version), Version,
		len(programID), programID,
		len(programVersion), programVersion,
	))
	for _, q := range qsos {
		writeField(&b, "CALL", q.Call)
		writeField(&b, "QSO_DATE", q.QsoDate)
		writeField(&b, "TIME_ON", q.TimeOn)
		if q.TimeOff != "" {
			writeField(&b, "TIME_OFF", q.TimeOff)
		}
		if q.Band != "" {
			writeField(&b, "BAND", strings.ToLower(q.Band))
		}
		if q.FreqMHz != 0 {
			writeField(&b, "FREQ", strconv.FormatFloat(q.FreqMHz, 'f', -1, 64))
		}
		writeField(&b, "MODE", q.Mode)
		if q.Submode != "" {
			writeField(&b, "SUBMODE", q.Submode)
		}
		writeField(&b, "RST_SENT", q.RstSent)
		writeField(&b, "RST_RCVD", q.RstRcvd)
		if q.Name != "" {
			writeField(&b, "NAME", q.Name)
		}
		if q.Qth != "" {
			writeField(&b, "QTH", q.Qth)
		}
		if q.Comment != "" {
			writeField(&b, "COMMENT", q.Comment)
		}
		if q.TxPower != 0 {
			writeField(&b, "TX_PWR", strconv.FormatFloat(q.TxPower, 'f', -1, 64))
		}
		if q.MyCall != "" {
			writeField(&b, "STATION_CALLSIGN", q.MyCall)
		}
		if q.MyGrid != "" {
			writeField(&b, "MY_GRIDSQUARE", q.MyGrid)
		}
		if q.MySigInfo != "" {
			writeField(&b, "MY_SIG_INFO", q.MySigInfo)
		}
		if q.SigInfo != "" {
			writeField(&b, "SIG_INFO", q.SigInfo)
		}
		if q.SotaRef != "" {
			writeField(&b, "SOTA_REF", q.SotaRef)
		}
		if q.MySotaRef != "" {
			writeField(&b, "MY_SOTA_REF", q.MySotaRef)
		}
		b.WriteString("<EOR>\n")
	}
	return b.String()
}

func writeField(b *strings.Builder, name, value string) {
	fmt.Fprintf(b, "<%s:%d>%s\n", name, len(value), value)
}

// ImportResult tallies an import.
type ImportResult struct {
	QSOs    []qso.QSO
	Skipped int
	Errors  []string
}

// Import parses an .adi document. Unknown fields are stashed in RawADIF.
// Records failing validation are skipped and counted.
func Import(data string) ImportResult {
	var res ImportResult
	// Strip header up to <EOH>
	upper := strings.ToUpper(data)
	if i := strings.Index(upper, "<EOH>"); i >= 0 {
		data = data[i + len("<EOH>"):]
	}
	// Split records on <EOR> (case-insensitive)
	records := regexp.MustCompile(`(?i)<EOR>`).Split(data, -1)
	for idx, rec := range records {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		fields := map[string]string{}
		var unknown []string
		for _, m := range fieldRe.FindAllStringSubmatch(rec, -1) {
			name := strings.ToUpper(m[1])
			val := strings.TrimSpace(m[3])
			fields[name] = val
			switch name {
			case "CALL", "QSO_DATE", "TIME_ON", "TIME_OFF", "BAND", "FREQ",
				"MODE", "SUBMODE", "RST_SENT", "RST_RCVD", "NAME", "QTH",
				"COMMENT", "TX_PWR", "STATION_CALLSIGN", "MY_CALL",
				"MY_GRIDSQUARE", "MY_SIG_INFO", "SIG_INFO",
				"SOTA_REF", "MY_SOTA_REF", "MY_POTA_REF", "POTA_REF",
				"ADIF_VER", "PROGRAMID", "PROGRAMVERSION", "EOR", "EOH":
				// known
			default:
				unknown = append(unknown, m[0])
			}
		}
		if _, ok := fields["CALL"]; !ok {
			// empty chunk (e.g. trailing) — skip silently only if no fields
			if len(fields) == 0 {
				continue
			}
			res.Skipped++
			res.Errors = append(res.Errors, fmt.Sprintf("record %d: missing CALL", idx+1))
			continue
		}
		q := qso.QSO{
			Call:      fields["CALL"],
			QsoDate:   fields["QSO_DATE"],
			TimeOn:    fields["TIME_ON"],
			TimeOff:   fields["TIME_OFF"],
			Band:      strings.ToLower(fields["BAND"]),
			Mode:      strings.ToUpper(fields["MODE"]),
			Submode:   strings.ToUpper(fields["SUBMODE"]),
			RstSent:   fields["RST_SENT"],
			RstRcvd:   fields["RST_RCVD"],
			Name:      fields["NAME"],
			Qth:       fields["QTH"],
			Comment:   fields["COMMENT"],
			MySigInfo: fields["MY_SIG_INFO"],
			SigInfo:   fields["SIG_INFO"],
			SotaRef:   fields["SOTA_REF"],
			MySotaRef: fields["MY_SOTA_REF"],
			RawADIF:   strings.Join(unknown, " "),
		}
		// POTA aliases
		if q.MySigInfo == "" {
			q.MySigInfo = fields["MY_POTA_REF"]
		}
		if q.SigInfo == "" {
			q.SigInfo = fields["POTA_REF"]
		}
		if v := fields["FREQ"]; v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				q.FreqMHz = f
			}
		}
		if v := fields["TX_PWR"]; v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				q.TxPower = f
			}
		}
		if v := fields["STATION_CALLSIGN"]; v != "" {
			q.MyCall = v
		} else if v := fields["MY_CALL"]; v != "" {
			q.MyCall = v
		}
		if v := fields["MY_GRIDSQUARE"]; v != "" {
			q.MyGrid = v
		}
		if err := q.Validate(); err != nil {
			res.Skipped++
			res.Errors = append(res.Errors, fmt.Sprintf("record %d (%s): %v", idx+1, q.Call, err))
			continue
		}
		res.QSOs = append(res.QSOs, q)
	}
	return res
}
