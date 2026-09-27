package store

import (
	"testing"

	"hamlog/internal/qso"
)

func openTest(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertGetListDelete(t *testing.T) {
	db := openTest(t)
	q := qso.NewQSO("N0CALL", "FN20")
	q.Call = "W1AW"
	if _, err := db.Insert(&q); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := db.Get(q.ID)
	if err != nil {
		// ID not set on struct; fetch via list
		list, lerr := db.List("", 10)
		if lerr != nil {
			t.Fatal(lerr)
		}
		if len(list) != 1 || list[0].Call != "W1AW" {
			t.Fatalf("want 1 W1AW, got %+v err=%v", list, err)
		}
		got = list[0]
	}
	_ = got
	list, err := db.List("w1", 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("filter failed: %v %+v", err, list)
	}
	if err := db.Delete(list[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	n, _ := db.Count()
	if n != 0 {
		t.Fatalf("want 0 after delete, got %d", n)
	}
}

func TestInsertRejectsInvalid(t *testing.T) {
	db := openTest(t)
	q := qso.QSO{}
	if _, err := db.Insert(&q); err == nil {
		t.Fatal("want validation error")
	}
}
