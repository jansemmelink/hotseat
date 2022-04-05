package db_test

import (
	"encoding/json"
	"testing"
	"time"

	"bitbucket.org/vservices/hotseat/db"
)

func TestSqlTime(t *testing.T) {
	T1 := "2022-04-01 10:11:12"
	expValue := []uint8{50, 48, 50, 50, 45, 48, 52, 45, 48, 49, 32, 49, 48, 58, 49, 49, 58, 49, 50}

	t1, _ := time.Parse("2006-01-02 15:04:05", T1)

	s1 := db.SqlTime(t1)
	v1, _ := s1.Value()
	if _, ok := v1.(string); !ok {
		t.Fatalf("Value -> %T != string", v1)
	}
	t.Logf("v1:       (%T) %+v", v1, v1)
	t.Logf("expValue: (%T) %+v", expValue, expValue)
	b1 := []byte(v1.(string))
	if len(b1) != len(expValue) {
		t.Fatalf("len %d != %d", len(b1), len(expValue))
	}
	for i, b := range expValue {
		if b1[i] != b {
			t.Fatalf("byte[%d] %v != %v", i, b, b1[i])
		}
	}
	j1, err := json.Marshal(s1)
	if err != nil {
		t.Fatal(err)
	}
	if string(j1) != "\""+T1+"\"" {
		t.Fatalf("json %s != %s", string(j1), T1)
	}

	var s2 db.SqlTime
	if err := json.Unmarshal([]byte("\""+T1+"\""), &s2); err != nil {
		t.Fatal(err)
	}
	if s2 != s1 {
		t.Fatalf("%v != %v", s1, s2)
	}

	v1, _ = s1.Value()
	b1 = []byte(v1.(string))
	if err := s2.Scan(b1); err != nil {
		t.Fatal(err)
	}
	if s2 != s1 {
		t.Fatalf("%v != %v", s1, s2)
	}
}
