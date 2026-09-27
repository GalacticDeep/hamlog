package store

import (
	"database/sql"
	"strings"
	"time"

	"hamlog/internal/qso"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS qsos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  call TEXT NOT NULL,
  qso_date TEXT NOT NULL,
  time_on TEXT NOT NULL,
  time_off TEXT NOT NULL DEFAULT '',
  band TEXT NOT NULL DEFAULT '',
  freq_mhz REAL NOT NULL DEFAULT 0,
  mode TEXT NOT NULL DEFAULT '',
  submode TEXT NOT NULL DEFAULT '',
  rst_sent TEXT NOT NULL DEFAULT '',
  rst_rcvd TEXT NOT NULL DEFAULT '',
  name TEXT NOT NULL DEFAULT '',
  qth TEXT NOT NULL DEFAULT '',
  comment TEXT NOT NULL DEFAULT '',
  tx_power REAL NOT NULL DEFAULT 0,
  my_call TEXT NOT NULL DEFAULT '',
  my_grid TEXT NOT NULL DEFAULT '',
  my_sig_info TEXT NOT NULL DEFAULT '',
  sig_info TEXT NOT NULL DEFAULT '',
  sota_ref TEXT NOT NULL DEFAULT '',
  my_sota_ref TEXT NOT NULL DEFAULT '',
  raw_adif TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_qsos_call ON qsos(call);
CREATE INDEX IF NOT EXISTS idx_qsos_date ON qsos(qso_date, time_on);
`

// DB wraps sql.DB.
type DB struct {
	sql *sql.DB
}

// Open opens (creating dirs implied by caller) and migrates.
func Open(path string) (*DB, error) {
	sdb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := sdb.Exec(schema); err != nil {
		sdb.Close()
		return nil, err
	}
	return &DB{sql: sdb}, nil
}

// Close closes the DB.
func (d *DB) Close() error { return d.sql.Close() }

func scanRow(scanner interface {
	Scan(dest ...any) error
}, q *qso.QSO) error {
	var created, updated string
	err := scanner.Scan(
		&q.ID, &q.Call, &q.QsoDate, &q.TimeOn, &q.TimeOff,
		&q.Band, &q.FreqMHz, &q.Mode, &q.Submode,
		&q.RstSent, &q.RstRcvd, &q.Name, &q.Qth, &q.Comment,
		&q.TxPower, &q.MyCall, &q.MyGrid, &q.MySigInfo, &q.SigInfo,
		&q.SotaRef, &q.MySotaRef, &q.RawADIF, &created, &updated,
	)
	if err != nil {
		return err
	}
	if t, err := time.Parse(time.RFC3339, created); err == nil {
		q.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updated); err == nil {
		q.UpdatedAt = t
	}
	return nil
}

// Insert validates and inserts a QSO.
func (d *DB) Insert(q *qso.QSO) (int64, error) {
	if err := q.Validate(); err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	q.CreatedAt = time.Now().UTC()
	q.UpdatedAt = q.CreatedAt
	res, err := d.sql.Exec(`INSERT INTO qsos
	(call,qso_date,time_on,time_off,band,freq_mhz,mode,submode,rst_sent,rst_rcvd,
	 name,qth,comment,tx_power,my_call,my_grid,my_sig_info,sig_info,sota_ref,my_sota_ref,
	 raw_adif,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		q.Call, q.QsoDate, q.TimeOn, q.TimeOff, q.Band, q.FreqMHz, q.Mode, q.Submode,
		q.RstSent, q.RstRcvd, q.Name, q.Qth, q.Comment, q.TxPower,
		q.MyCall, q.MyGrid, q.MySigInfo, q.SigInfo, q.SotaRef, q.MySotaRef,
		q.RawADIF, now, now,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	q.ID = id
	return id, nil
}

// Update validates and updates a QSO by ID.
func (d *DB) Update(q *qso.QSO) error {
	if err := q.Validate(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.sql.Exec(`UPDATE qsos SET
	 call=?,qso_date=?,time_on=?,time_off=?,band=?,freq_mhz=?,mode=?,submode=?,
	 rst_sent=?,rst_rcvd=?,name=?,qth=?,comment=?,tx_power=?,my_call=?,my_grid=?,
	 my_sig_info=?,sig_info=?,sota_ref=?,my_sota_ref=?,raw_adif=?,updated_at=?
	 WHERE id=?`,
		q.Call, q.QsoDate, q.TimeOn, q.TimeOff, q.Band, q.FreqMHz, q.Mode, q.Submode,
		q.RstSent, q.RstRcvd, q.Name, q.Qth, q.Comment, q.TxPower,
		q.MyCall, q.MyGrid, q.MySigInfo, q.SigInfo, q.SotaRef, q.MySotaRef,
		q.RawADIF, now, q.ID,
	)
	return err
}

// Delete removes a QSO by ID.
func (d *DB) Delete(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM qsos WHERE id=?`, id)
	return err
}

// Get fetches one QSO.
func (d *DB) Get(id int64) (qso.QSO, error) {
	var q qso.QSO
	row := d.sql.QueryRow(`SELECT id,call,qso_date,time_on,time_off,band,freq_mhz,
		mode,submode,rst_sent,rst_rcvd,name,qth,comment,tx_power,my_call,my_grid,
		my_sig_info,sig_info,sota_ref,my_sota_ref,raw_adif,created_at,updated_at
		FROM qsos WHERE id=?`, id)
	if err := scanRow(row, &q); err != nil {
		return q, err
	}
	return q, nil
}

// List returns QSOs newest-first, optionally filtered by call substring.
func (d *DB) List(filter string, limit int) ([]qso.QSO, error) {
	var rows *sql.Rows
	var err error
	if limit <= 0 {
		limit = 1000
	}
	if f := strings.ToUpper(strings.TrimSpace(filter)); f != "" {
		rows, err = d.sql.Query(`SELECT id,call,qso_date,time_on,time_off,band,freq_mhz,
			mode,submode,rst_sent,rst_rcvd,name,qth,comment,tx_power,my_call,my_grid,
			my_sig_info,sig_info,sota_ref,my_sota_ref,raw_adif,created_at,updated_at
			FROM qsos WHERE call LIKE ? ORDER BY qso_date DESC, time_on DESC LIMIT ?`, "%"+f+"%", limit)
	} else {
		rows, err = d.sql.Query(`SELECT id,call,qso_date,time_on,time_off,band,freq_mhz,
			mode,submode,rst_sent,rst_rcvd,name,qth,comment,tx_power,my_call,my_grid,
			my_sig_info,sig_info,sota_ref,my_sota_ref,raw_adif,created_at,updated_at
			FROM qsos ORDER BY qso_date DESC, time_on DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []qso.QSO
	for rows.Next() {
		var q qso.QSO
		if err := scanRow(rows, &q); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// Count returns total QSO count.
func (d *DB) Count() (int, error) {
	var n int
	err := d.sql.QueryRow(`SELECT COUNT(*) FROM qsos`).Scan(&n)
	return n, err
}
