package db_test

import (
	"testing"

	"bitbucket.org/vservices/hotseat/db"
)

func TestEmailPattern(t *testing.T) {
	validEmails := []string{
		"a@b.c",
		"a_b@c.d",
		"a-b@c.d",
		"a@b-c.d",
		"a@b_c.d",
		"a@b.c.d",
		"a@b_c.d-e.f",
		"jan.semmelink@gmail.com",
	}
	for i, e := range validEmails {
		if db.ValidEmail(e) {
			t.Logf("[%3d] OK Valid \"%s\"", i, e)
		} else {
			t.Errorf("[%3d] ERROR \"%s\" indicate invalid", i, e)
		}
	}

	invalidEmails := []string{
		"a@b.",
		"@b.c",
		"@b",
		"@.",
		"a-@b.c",
		"a-.b@c.d",
		"jan.semmelink@gmail",
	}
	for i, e := range invalidEmails {
		if !db.ValidEmail(e) {
			t.Logf("[%3d] OK Invalid \"%s\"", i, e)
		} else {
			t.Errorf("[%3d] ERROR \"%s\" indicate valid", i, e)
		}
	}
}
