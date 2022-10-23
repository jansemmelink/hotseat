package db_test

import (
	"testing"

	"bitbucket.org/vservices/hotseat/db"
)

func TestMetas(t *testing.T) {
	if err := db.SetMetas("groups", "12345", db.Metas{
		"a": "1",
		"b": "2",
	}); err != nil {
		t.Fatalf("failed to set metas: %+v", err)
	}

	if err := db.SetMetas("groups", "12346", db.Metas{
		"a": "5",
		"b": "6",
	}); err != nil {
		t.Fatalf("failed to set metas: %+v", err)
	}

	if err := db.SetMetas("groups", "12345", db.Metas{
		"a": "3",
		"b": "2",
		"c": "4",
	}); err != nil {
		t.Fatalf("failed to set metas: %+v", err)
	}

	metas, err := db.GetMetas("groups", "12345")
	if err != nil {
		t.Fatalf("failed to get metas: %+v", err)
	}
	if metas["a"] != "3" || metas["b"] != "2" || metas["c"] != "4" || len(metas) != 3 {
		t.Fatalf("wrong metas: %+v", metas)
	}

	if err := db.DelMetas("groups", "12345", []string{"b", "a"}); err != nil {
		t.Fatalf("failed to delete named metas")
	}

	metas, err = db.GetMetas("groups", "12345")
	if err != nil {
		t.Fatalf("failed to get metas: %+v", err)
	}
	if metas["c"] != "4" || len(metas) != 1 { //a and b was deleted
		t.Fatalf("wrong metas: %+v", metas)
	}

	metas, err = db.GetMetas("groups", "12346")
	if err != nil {
		t.Fatalf("failed to get metas: %+v", err)
	}
	if metas["a"] != "5" || metas["b"] != "6" || len(metas) != 2 {
		t.Fatalf("wrong metas: %+v", metas)
	}

	for _, id := range []string{"12345", "12346", "12347"} {
		if err := db.DelAllMetas("groups", id); err != nil {
			t.Fatalf("failed to delete metas for group %s: %+v", id, err)
		}
	}

	metas, err = db.GetMetas("groups", "12345")
	if err != nil {
		t.Fatalf("failed to get metas: %+v", err)
	}
	if len(metas) != 0 {
		t.Fatalf("wrong metas: %+v", metas)
	}

	metas, err = db.GetMetas("groups", "12346")
	if err != nil {
		t.Fatalf("failed to get metas: %+v", err)
	}
	if len(metas) != 0 {
		t.Fatalf("wrong metas: %+v", metas)
	}
}
