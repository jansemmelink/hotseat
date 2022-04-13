package db

import (
	"time"

	"github.com/go-msvc/errors"
	"github.com/google/uuid"
)

type Account struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Active bool       `json:"active"`
	Admin  bool       `json:"admin,omitempty"`
	Expiry *time.Time `json:"expiry"`
}

type AccountsFilter struct {
	ID     *string //full id or nothing
	Name   *string //part of name
	Active *bool
	Admin  *bool
}

func GetAccounts(filter AccountsFilter, sort []string, limit int) ([]Account, error) {
	log.Debugf("GetAccounts(filter: %+v, sort: %+v, limit: %d)", filter, sort, limit)
	var accounts []Account
	if err := FilteredSelect(
		&accounts,
		"SELECT * FROM accounts",
		mapValues(filter),
		limit,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to read accounts")
	}
	return accounts, nil
}

type NewAccount struct {
	Name       string `json:"name"`
	AdminEmail string `json:"admin_email"`
}

func AddAccount(newAccount NewAccount) (accountAdminUser *User, activationToken string, err error) {
	if newAccount.Name == "" {
		return nil, "", errors.Errorf("missing name")
	}
	if newAccount.AdminEmail == "" {
		return nil, "", errors.Errorf("missing admin_email")
	}
	if !ValidEmail(newAccount.AdminEmail) {
		return nil, "", errors.Errorf("admin_email:\"%s\" is not a valid email address", newAccount.AdminEmail)
	}
	accountAdminUser = &User{Account: &Account{}}
	accountAdminUser.Account.ID = uuid.New().String()
	accountAdminUser.Account.Name = newAccount.Name
	accountAdminUser.Account.Admin = false //not system admin account
	accountAdminUser.Account.Active = true
	accountAdminUser.Account.Expiry = nil
	if _, err := db.NamedExec(
		"INSERT INTO accounts SET id=:id,name=:name,admin=false,active=true,expiry=null",
		map[string]interface{}{
			"id":   accountAdminUser.Account.ID,
			"name": newAccount.Name,
		},
	); err != nil {
		return nil, "", errors.Wrapf(err, "failed to create account")
	}

	accountAdminUser.ID = uuid.New().String()
	accountAdminUser.Username = newAccount.AdminEmail
	accountAdminUser.Expiry = nil
	accountAdminUser.Active = false
	accountAdminUser.Admin = true //account admin user
	activationToken = passwordHash(*accountAdminUser, newRandomPassword(10))
	if _, err := db.NamedExec(
		"INSERT INTO users SET id=:id,account_id=:account_id,username=:username,passhash=:passhash,admin=true,active=false,expiry=null",
		map[string]interface{}{
			"id":         accountAdminUser.ID,
			"account_id": accountAdminUser.Account.ID,
			"username":   accountAdminUser.Username,
			"passhash":   activationToken,
		},
	); err != nil {
		return nil, "", errors.Wrapf(err, "failed to create account admin user")
	}
	return accountAdminUser, activationToken, nil
}

func GetAccount(accountID string) (*Account, error) {
	var account Account
	if err := NamedGet(
		&account,
		"SELECT * from accounts where id=:id",
		map[string]interface{}{
			"id": accountID,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get account")
	}
	return &account, nil
}

func GetAccountByName(name string) (*Account, error) {
	var account Account
	if err := NamedGet(
		&account,
		"SELECT * from accounts where name=:name",
		map[string]interface{}{
			"name": name,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get account")
	}
	return &account, nil
}

var publicAccount *Account
