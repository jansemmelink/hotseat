package db

import (
	"time"

	"github.com/go-msvc/errors"
	"github.com/google/uuid"
)

//groups of any members in the db
type Group struct {
	ID        string   `json:"id"`
	Account   *Account `json:"account,omitempty"`
	OwnerType string   `json:"owner_type"` //e.g. user or account or ...
	OwnerID   string   `json:"owner_id"`   //e.g. user.id or account.id or ...
	Name      string   `json:"name"`
}

type GroupRow struct {
	ID            string   `db:"id"`
	AccountID     string   `db:"account_id"`
	AccountName   string   `db:"account_name"`
	AccountActive bool     `db:"account_active"`
	AccountAdmin  bool     `db:"account_admin"`
	Expiry        *SqlTime `db:"account_expiry"`
	OwnerType     string   `db:"owner_type"` //e.g. user or account	(could also be a group of xxx)
	OwnerID       string   `db:"owner_id"`   //e.g. user.id or account.id
	Name          string   `db:"name"`
}

type GroupMember struct {
	Group      *Group `json:"group,omitempty"`
	MemberType string `json:"member_type"` //e.g. user or xxx
	MemberID   string `json:"member_id"`   //id of the member in its table
	//Values map[string]string `json:"values,omitempty"`
}

type GroupsFilter struct {
	ID        *string `db:"id"` //id to match full or any id
	AccountID *string `db:"account_id"`
	OwnerType *string `db:"owner_type"`
	OwnerID   *string `db:"owner_id"`
	Name      *string `db:"name"` //part of name or else any name
}

func GetGroups(filter GroupsFilter, sort []string, limit int) ([]Group, error) {
	log.Debugf("GetGroups(filter: %+v, sort: %+v, limit: %d)", filter, sort, limit)
	var groupRows []GroupRow
	if err := FilteredSelect(
		&groupRows,
		"SELECT g.id,a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin,a.expiry as account_expiry,g.name,g.owner_type,g.owner_id FROM groups as g INNER JOIN accounts as a on a.id=g.account_id",
		mapValues(filter),
		limit,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get groups")
	}
	g := make([]Group, len(groupRows))
	for i, gr := range groupRows {
		g[i] = Group{
			ID: gr.ID,
			Account: &Account{
				ID:     gr.AccountID,
				Name:   gr.AccountName,
				Admin:  gr.AccountAdmin,
				Active: gr.AccountActive,
				Expiry: (*time.Time)(gr.Expiry),
			},
			OwnerType: gr.OwnerType,
			OwnerID:   gr.OwnerID,
			Name:      gr.Name,
		}
	}
	return g, nil
}

func AddGroup(g Group) (*Group, error) {
	if g.ID != "" {
		return nil, errors.Errorf("id specified for add")
	}
	if g.Account == nil || g.Account.ID == "" {
		return nil, errors.Errorf("missing account")
	}
	if g.Name == "" {
		return nil, errors.Errorf("missing name")
	}

	//check that owner_type+owner_id refers to existing item in the db
	var row ItemRow
	if err := NamedGet(&row, "SELECT id FROM "+g.OwnerType+"s WHERE id=:id", map[string]interface{}{
		"id": g.OwnerID,
	}); err != nil {
		return nil, errors.Errorf("group owner not found %s:{\"id\":\"%s\"}", g.OwnerType, g.OwnerID)
	}

	//create the group
	g.ID = uuid.New().String()
	if _, err := db.NamedExec(
		"insert into groups set id=:id,account_id=:account_id,owner_type=:owner_type,owner_id=:owner_id,name=:name",
		map[string]interface{}{
			"id":         g.ID,
			"account_id": g.Account.ID,
			"owner_type": g.OwnerType,
			"owner_id":   g.OwnerID,
			"name":       g.Name,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to insert group")
	}
	return &g, nil
} //AddGroup()

func GetGroup(id string) (*Group, error) {
	gr := GroupRow{
		ID: id,
	}
	if err := NamedGet(
		&gr,
		"SELECT g.id,a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin,a.expiry as account_expiry,g.name,g.owner_type,g.owner_id FROM groups as g INNER JOIN accounts as a on a.id=g.account_id WHERE g.id=:id",
		map[string]interface{}{
			"id": id,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get group")
	}
	return &Group{}, nil
} //GetGroup()

type GroupMembersFilter struct {
}

func GetGroupMembers(id string, filter GroupMembersFilter, sort []string, limit int) ([]GroupMember, error) {
	log.Debugf("GetGroupMembers(id: %s, filter: %+v, sort: %+v, limit: %d)", id, filter, sort, limit)
	var userGroupMembers []GroupMember
	if err := FilteredSelect(
		&userGroupMembers,
		"SELECT * FROM group_members",
		mapValues(filter),
		limit,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to read group_members")
	}
	return userGroupMembers, nil
} //GetGroupMembers()

func AddGroupMember(groupID string, memberType string, memberID string) (*GroupMember, error) {
	g, err := GetGroup(groupID)
	if err != nil {
		return nil, errors.Errorf("group(%s) not found", groupID)
	}
	var row AccountItemRow
	if err := NamedGet(&row, "SELECT id,account_id FROM "+memberType+"s WHERE id=:id", map[string]interface{}{"id": memberID}); err != nil {
		return nil, errors.Errorf("cannot get member %s:{\"id\":\"%s\"}", memberType, memberID)
	}
	if row.AccountID != g.Account.ID {
		return nil, errors.Errorf("cannot add %s from other account", memberType)
	}

	id := uuid.New().String()
	if _, err := db.NamedExec(
		"insert into group_members set id=:id,group_id=:group_id,member_type=:member_type,member_id=:member_id",
		map[string]interface{}{
			"id":          id,
			"group_id":    groupID,
			"member_type": memberType,
			"member_id":   memberID,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to insert group_member")
	}
	return &GroupMember{
		Group:      g,
		MemberType: memberType,
		MemberID:   memberID,
	}, nil
} //AddGroupMember()

func DelGroupMember(gid string, uid string) error {
	stmt, err := getCompiledStatement("DELETE FROM group_members WHERE group_id=:group_id,user_id=:user_id")
	if err != nil {
		return errors.Errorf("failed to prepare")
	}
	if _, err := stmt.Exec(
		map[string]interface{}{
			"group_id": gid,
			"user_id":  uid,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to delete group_member")
	}
	return nil
} //DelGroupMember()
