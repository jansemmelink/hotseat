package db

import (
	"math/rand"
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
	Name string `json:"name"`
}

func AddAccount(newAccount NewAccount) (accountAdminUser *User, password string, err error) {
	if newAccount.Name == "" {
		return nil, "", errors.Errorf("missing name")
	}
	accountAdminUser = &User{Account: Account{}}
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
	accountAdminUser.Username = accountAdminUser.Account.Name + ".admin"
	accountAdminUser.Expiry = nil
	accountAdminUser.Active = true
	accountAdminUser.Admin = true //account admin user
	password = newRandomPassword(10)
	passhash := passwordHash(*accountAdminUser, password)
	if _, err := db.NamedExec(
		"INSERT INTO users SET id=:id,account_id=:account_id,username=:username,passhash=:passhash,admin=true,active=true,expiry=null",
		map[string]interface{}{
			"id":         accountAdminUser.ID,
			"account_id": accountAdminUser.Account.ID,
			"username":   accountAdminUser.Username,
			"passhash":   passhash,
		},
	); err != nil {
		return nil, "", errors.Wrapf(err, "failed to create account admin user")
	}
	return accountAdminUser, password, nil
}

func newRandomPassword(n int) string {
	s := ""
	c := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*()_+-={}[]:\";'\\|<>,.?/"
	for i := 0; i < n; i++ {
		s += string(c[rand.Intn(len(c))])
	}
	return s
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
