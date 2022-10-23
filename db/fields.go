package db

import (
	"fmt"
	"strings"

	"github.com/go-msvc/errors"
)

type Field struct {
	TableName   string `json:"table_name"`
	TableID     string `json:"table_id"`
	OrderNr     int    `json:"order_nr"  doc:"Ordering number, any int, fields are sorted in ascending order, duplicates allowed then order can vary among those fields"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type FieldRow struct {
	TableName   string `db:"table_name"`
	TableID     string `db:"table_id"`
	OrderNr     int    `db:"order_nr"`
	Name        string `db:"name"`
	Type        string `db:"type"`
	Description string `db:"description"`
}

func GetFields(tableName string, tableID string) ([]Field, error) {
	rows := []FieldRow{}
	if err := db.Select(
		&rows,
		"SELECT order_nr,name,type,description FROM `fields` WHERE table_name=? AND table_id=? ORDER BY `order_nr`,`name`",
		tableName,
		tableID,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to select fields")
	}

	fields := []Field{}
	for _, r := range rows {
		fields = append(fields, Field{
			TableName:   tableName,
			TableID:     tableID,
			OrderNr:     r.OrderNr,
			Name:        r.Name,
			Type:        r.Type,
			Description: r.Description,
		})
	}
	return fields, nil
} //GetFields()

func SetFields(tableName string, tableID string, fields []Field) error {
	for _, f := range fields {
		if _, err := db.NamedExec(
			"insert into `fields` set table_name=:table_name,table_id=:table_id,order_nr=:order_nr,name=:name,type=:type,description=:description ON DUPLICATE KEY UPDATE order_nr=:order_nr,type=:type,description=:description",
			map[string]interface{}{
				"table_name":  tableName,
				"table_id":    tableID,
				"order_nr":    f.OrderNr,
				"name":        f.Name,
				"type":        f.Type,
				"description": f.Description,
			},
		); err != nil {
			return errors.Wrapf(err, "failed to set field %+v", f)
		}
	}
	return nil
} //SetFields()

func DelAllFields(tableName string, tableID string) error {
	if _, err := db.NamedExec(
		"DELETE FROM `fields` WHERE table_name=:table_name AND table_id=:table_id",
		map[string]interface{}{
			"table_name": tableName,
			"table_id":   tableID,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to delete fields")
	}
	return nil
} //DelAllFields()

func DelFields(tableName string, tableID string, names []string) error {
	if len(names) < 1 {
		return nil
	}
	query := "DELETE FROM `fields` WHERE table_name=:table_name AND table_id=:table_id AND ("
	args := map[string]interface{}{
		"table_name": tableName,
		"table_id":   tableID,
	}
	for i, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			return errors.Errorf("empty param name")
		}
		p := fmt.Sprintf("n%d", i)
		if i > 0 {
			query += " OR "
		}
		query += "name=:" + p
		args[p] = n
	}
	query += ")"
	if _, err := db.NamedExec(query, args); err != nil {
		return errors.Wrapf(err, "failed to delete named fields")
	}
	return nil
} //DelFields()
