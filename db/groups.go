package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-msvc/errors"
	"github.com/google/uuid"
)

//group of persons that are called members
//group belongs to an account (because account can pay for benefit to have up to N groups)
//only account admin can create/manage groups
//group can have sub-groups and those may be created by other accounts allowed to do so
type Group struct {
	ID          string                 `json:"id"`
	Account     *Account               `json:"account"`
	Parent      *Group                 `json:"parent,omitempty"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Invitation  *bool                  `json:"invitation,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty" doc:"Group data, e.g. membership cost"`
}

type GroupRow struct {
	ID            string   `db:"id"`
	AccountID     string   `db:"account_id"`
	AccountName   string   `db:"account_name"`
	AccountActive bool     `db:"account_active"`
	AccountAdmin  bool     `db:"account_admin"`
	AccountExpiry *SqlTime `db:"account_expiry"`
	ParentGroupID *string  `db:"parent_group_id"`
	Name          string   `db:"name"`
	Description   *string  `db:"description"`
	Invitation    *bool    `db:"invitation"`
}

const queryGroup = "SELECT g.id,a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin,a.expiry as account_expiry,g.name,g.description,g.parent_group_id,g.invitation FROM groups as g INNER JOIN accounts as a on a.id=g.account_id"

type GroupMember struct {
	Group      *Group `json:"group,omitempty"`
	MemberType string `json:"member_type"` //e.g. user or xxx
	MemberID   string `json:"member_id"`   //id of the member in its table
	//Values map[string]string `json:"values,omitempty"`
}

type GroupsFilter struct {
	ID            *string `db:"id"` //id to match full or any id
	AccountID     *string `db:"account_id"`
	ParentGroupID *string `db:"parent_group_id"`
	Name          *string `db:"name"` //part of name or else any name
}

func GetGroups(filter GroupsFilter, sort []string, limit int) ([]Group, error) {
	log.Debugf("GetGroups(filter: %+v, sort: %+v, limit: %d)", filter, sort, limit)
	var groupRows []GroupRow
	if err := FilteredSelect(
		&groupRows,
		queryGroup,
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
				Expiry: (*time.Time)(gr.AccountExpiry),
			},
			Parent:      nil,
			Name:        gr.Name,
			Description: gr.Description,
		}
		if gr.ParentGroupID != nil && *gr.ParentGroupID != "" {
			parentGroup, err := GetGroup(*gr.ParentGroupID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get group.id:\"%s\".parent_group_id:\"%s\"", gr.ID, *gr.ParentGroupID)
			}
			g[i].Parent = parentGroup
		}
	}
	return g, nil
}

type NewGroup struct {
	ParentGroupID *string                `json:"parent_group_id" doc:"Parent only if creating a sub-group. To invite another account holder to create a sub to one of your groups, leave name and description empty and specify their account ID"`
	AccountID     *string                `json:"account_id" doc:"AccountID of another account invited to create this group (without name or description)"`
	Name          string                 `json:"name" doc:"Required name of the group, unique within scope of your account."`
	Description   *string                `json:"description" doc:"Optional description text"`
	Data          map[string]interface{} `json:"data" doc:"Additional data values for this group"`
}

func (ng *NewGroup) Validate() error {
	//if account is specified, it may not be yours, and this is only invitation to another account to
	//create a child group, and you may not specify name or description either but must specify parent group
	if ng.AccountID != nil && *ng.AccountID != "" {
		if ng.Name != "" || (ng.Description != nil && *ng.Description != "") {
			return errors.Errorf("name and description specified for sub group invitation")
		}
		if ng.ParentGroupID == nil || *ng.ParentGroupID == "" {
			return errors.Errorf("parent_group_id required for sub group invitation")
		}
	} else {
		ng.Name = strings.TrimSpace(ng.Name)
		if ng.Name == "" {
			return errors.Errorf("missing name")
		}
		if ng.Description != nil {
			*ng.Description = strings.TrimSpace(*ng.Description)
			if *ng.Description == "" {
				ng.Description = nil
			}
		}
	}
	if err := validateData(ng.Data); err != nil {
		return errors.Wrapf(err, "invalid data")
	}
	return nil
}

func AddGroup(user User, ng NewGroup) (*Group, error) {
	if !user.Admin {
		return nil, errors.Errorf("cannot add a group because user is not account admin user")
	}
	if err := ng.Validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid request")
	}

	//todo: check if allowed for this account...
	invitation := false
	var otherAccount *Account
	var otherUser *User
	if ng.AccountID != nil && *ng.AccountID != "" {
		if *ng.AccountID == user.Account.ID {
			return nil, errors.Errorf("account_id must be different from your own")
		}

		var err error
		otherAccount, err = GetAccount(*ng.AccountID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get account.id=%s", *ng.AccountID)
		}
		if users, err := GetUsers(map[string]interface{}{"account_id": otherAccount.ID, "admin": true}, nil, 1); err != nil || len(users) != 1 {
			return nil, errors.Errorf("failed to get other account admin user: %+v", err)
		} else {
			u := users[0]
			otherUser = &u
		}

		//parent group must belong to account
		var parentGroupRow GroupRow
		if err := NamedGet(
			&parentGroupRow,
			queryGroup+" WHERE g.id=:id && a.id=:account_id",
			map[string]interface{}{
				"id":         *ng.ParentGroupID,
				"account_id": user.Account.ID,
			},
		); err != nil {
			return nil, errors.Wrapf(err, "parent group.id=%s not found for your account.id=%s", *ng.ParentGroupID, user.Account.ID)
		}
		invitation = true
		//use same name and description as parent group
		ng.Name = parentGroupRow.Name
		ng.Description = parentGroupRow.Description
	} else {
		ng.AccountID = &user.Account.ID
	}

	//create the group
	id := uuid.New().String()
	params := map[string]interface{}{
		"id":          id,
		"aid":         ng.AccountID,
		"pgid":        ng.ParentGroupID,
		"name":        ng.Name,
		"description": ng.Description,
		"invitation":  invitation,
	}
	if _, err := db.NamedExec(
		"insert into groups set id=:id,account_id=:aid,parent_group_id=:pgid,name=:name,description=:description,invitation=:invitation",
		params,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to create group")
	}

	if len(ng.Data) > 0 {
		if err := SetMetas("groups", id, ng.Data); err != nil {
			return nil, errors.Wrapf(err, "failed to store group data")
		}
	}

	//get all details of the new group
	g, err := GetGroup(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get new group")
	}

	if invitation {
		//send message to other account user about the invitation
		messageID, err := user.SendMessage(
			otherUser,
			fmt.Sprintf("You are invited to create a new group inside %s", ng.Name))
		if err != nil {
			log.Errorf("Failed to send invitation message")
		} else {
			log.Debugf("Sent message %+v", messageID)
		}
	}

	return g, nil
} //AddGroup()

func GetGroup(id string) (*Group, error) {
	gr := GroupRow{
		ID: id,
	}
	if err := NamedGet(
		&gr,
		"SELECT g.id,"+
			"a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin,a.expiry as account_expiry,"+
			"g.name,g.description,g.parent_group_id,g.invitation FROM groups as g INNER JOIN accounts as a on a.id=g.account_id WHERE g.id=:id",
		map[string]interface{}{
			"id": id,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get group")
	}
	log.Debugf("GROUP ROW: %+v", gr)
	g := &Group{
		ID:          gr.ID,
		Name:        gr.Name,
		Description: gr.Description,
		Account: &Account{
			ID:     gr.AccountID,
			Name:   gr.AccountName,
			Active: gr.AccountActive,
			Admin:  gr.AccountAdmin,
			Expiry: (*time.Time)(gr.AccountExpiry),
		},
		Invitation: gr.Invitation,
		Data:       nil,
	}
	if gr.ParentGroupID != nil && *gr.ParentGroupID != "" {
		parentGroup, err := GetGroup(*gr.ParentGroupID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get group parent")
		}
		g.Parent = parentGroup
	}

	if metas, err := GetMetas("groups", id); err == nil && metas != nil {
		g.Data = metas
	}
	return g, nil
} //GetGroup()

//updates group name and description and specified data
func UpdGroup(user User, g Group) error {
	if g.Account.ID != user.Account.ID {
		return errors.Errorf("group does not belong to you")
	}
	if _, err := db.NamedExec(
		"UPDATE groups SET name=:name,description=:description WHERE id=:id AND account_id=:account_id",
		map[string]interface{}{
			"name":        g.Name,
			"description": g.Description,
			"id":          g.ID,
			"account_id":  g.Account.ID,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to update group")
	}
	dataToDelete := []string{}
	dataToSet := map[string]interface{}{}
	for n, v := range g.Data {
		if v == nil {
			dataToDelete = append(dataToDelete, n)
		} else {
			dataToSet[n] = v
		}
	}
	if len(dataToDelete) > 0 {
		if err := DelMetas("groups", g.ID, dataToDelete); err != nil {
			return errors.Wrapf(err, "failed to delete metas")
		}
	}
	if len(dataToSet) > 0 {
		if err := SetMetas("groups", g.ID, dataToSet); err != nil {
			return errors.Wrapf(err, "failed to set metas")
		}
	}
	return nil
} //UpdGroup()

func DelGroup(user User, id string) error {
	//note: foreign key prevent deletion of parent group with children
	result, err := db.NamedExec(
		"DELETE FROM groups WHERE id=:id AND account_id=:account_id",
		map[string]interface{}{
			"id":         id,
			"account_id": user.Account.ID,
		},
	)
	if err != nil {
		return errors.Errorf("failed to delete")
	}
	nr, err := result.RowsAffected()
	if nr != 1 {
		return errors.Errorf("deleted %d groups, not 1: %+v", err)
	}

	//metas has no foreign key - delete after group was deleted
	DelAllMetas("groups", id)
	return nil
}

func GetGroupFields(id string, includeParentFields bool) ([]Field, error) {
	fields, err := GetFields("groups", id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read group fields")
	}

	if !includeParentFields {
		return fields, nil
	}

	//see if group has parent
	var g struct {
		ParentGroupID string `db:"parent_group_id"`
	}
	if err := db.Get(&g, "SELECT parent_group_id FROM `groups` WHERE id=?", id); err != nil {
		return nil, errors.Wrapf(err, "failed to determing parent_group_id")
	}
	if g.ParentGroupID == "" {
		return fields, nil //no parent
	}

	pf, err := GetGroupFields(g.ParentGroupID, includeParentFields)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read parent fields")
	}
	return append(pf, fields...), nil
}

func SetGroupFields(user User, id string, fields []Field) error {
	g, err := GetGroup(id)
	if err != nil {
		return errors.Wrapf(err, "cannot get group")
	}
	if g.Account.ID != user.Account.ID {
		return errors.Errorf("group does not belong to you")
	}
	return SetFields("groups", id, fields)
}

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
	if err := NamedGet(
		&row,
		"SELECT id,account_id FROM "+memberType+"s WHERE id=:id",
		map[string]interface{}{
			"id": memberID,
		},
	); err != nil {
		return nil, errors.Errorf("cannot get member \"%s\":{\"id\":\"%s\"}", memberType, memberID)
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

func validateData(data map[string]interface{}) error {
	for n := range data {
		if n != strings.TrimSpace(n) {
			return errors.Errorf("spaces in name \"%s\"", n)
		}
		//todo: v must have SQL value and parse methods
	}
	return nil
}
