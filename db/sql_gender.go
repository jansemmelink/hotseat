package db

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/go-msvc/errors"
)

type SqlGender int

const (
	SqlGenderUndefined SqlGender = iota
	SqlGenderMale
	SqlGenderFemale
)

func (t *SqlGender) Scan(value interface{}) error {
	if byteArray, ok := value.([]uint8); ok {
		strValue := strings.ToLower(string(byteArray))
		switch strValue {
		case "":
			*t = SqlGenderUndefined
		case "male":
			*t = SqlGenderMale
		case "female":
			*t = SqlGenderFemale
		default:
			return errors.Errorf("invalid gender \"%s\" != male|female", strValue)
		}
		return nil
	}
	if value == nil {
		*t = SqlGenderUndefined
		return nil
	}
	return errors.Errorf("%T is not []uint8", value)
}

func (t SqlGender) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t SqlGender) String() string {
	switch t {
	case SqlGenderMale:
		return "male"
	case SqlGenderFemale:
		return "female"
	case SqlGenderUndefined:
		return ""
	default:
		return ""
	}
}

func (t *SqlGender) UnmarshalJSON(v []byte) error {
	s := string(v)
	if len(s) < 2 || !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
		return errors.Errorf("invalid gender string %s (expects quoted \"male|female\")", s)
	}
	return t.Scan(v[1 : len(v)-1])
}

func (t SqlGender) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", t.String())), nil
}
