package db

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/go-msvc/errors"
)

//dates are always stored at UTC with format 2006-01-02
type SqlDate time.Time

func (t *SqlDate) Scan(value interface{}) error {
	if byteArray, ok := value.([]uint8); ok {
		strValue := string(byteArray)
		timeValue, err := time.ParseInLocation("2006-01-02", strValue, time.UTC)
		if err != nil {
			return err
		}
		*t = SqlDate(timeValue)
		return nil
	}
	if value == nil {
		return nil
	}
	return errors.Errorf("%T is not []uint8", value)
}

func (t SqlDate) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t SqlDate) String() string {
	return time.Time(t).UTC().Format("2006-01-02")
}

func (t *SqlDate) UnmarshalJSON(v []byte) error {
	s := string(v)
	if len(s) < 2 || !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
		return errors.Errorf("invalid time string %s (expects quoted \"2006-01-02\")", s)
	}
	return t.Scan(v[1 : len(v)-1])
}

func (t SqlDate) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("\"%s\"", t.String())
	return []byte(s), nil
}
