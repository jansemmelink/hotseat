package db

import (
	"fmt"
	"strings"

	"github.com/go-msvc/errors"
)

type Metas map[string]interface{}

type MetaRow struct {
	TableName string `db:"table_name"`
	TableID   string `db:"table_id"`
	Name      string `db:"name"`
	Value     string `db:"value"`
}

func GetMetas(tableName string, tableID string) (Metas, error) {
	rows := []MetaRow{}
	if err := db.Select(
		&rows,
		"SELECT name,value FROM `metas` WHERE table_name=? AND table_id=?",
		tableName,
		tableID,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to select metas")
	}

	m := Metas{}
	for _, r := range rows {
		m[r.Name] = r.Value
	}
	return m, nil
}

func SetMetas(tableName string, tableID string, metas Metas) error {
	for n, v := range metas {
		if _, err := db.NamedExec(
			"insert into `metas` set table_name=:table_name,table_id=:table_id,name=:name,value=:value ON DUPLICATE KEY UPDATE value=:value",
			map[string]interface{}{
				"table_name": tableName,
				"table_id":   tableID,
				"name":       n,
				"value":      v,
			},
		); err != nil {
			return errors.Wrapf(err, "failed to set meta")
		}
	}
	return nil
}

func DelAllMetas(tableName string, tableID string) error {
	if _, err := db.NamedExec(
		"DELETE FROM `metas` WHERE table_name=:table_name AND table_id=:table_id",
		map[string]interface{}{
			"table_name": tableName,
			"table_id":   tableID,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to delete metas")
	}
	return nil
}

func DelMetas(tableName string, tableID string, names []string) error {
	if len(names) < 1 {
		return nil
	}
	query := "DELETE FROM `metas` WHERE table_name=:table_name AND table_id=:table_id AND ("
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
		return errors.Wrapf(err, "failed to delete named metas")
	}
	return nil
}
