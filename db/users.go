package db

import (
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/go-msvc/errors"
	"github.com/google/uuid"
)

type User struct {
	ID       string     `json:"id"`
	Account  Account    `json:"account"`
	Username string     `json:"username"`
	Admin    bool       `json:"admin,omitempty"`
	Active   bool       `json:"active"`
	Expiry   *time.Time `json:"expiry,omitempty"`
}

func GetUsers(filter map[string]interface{}, sort []string, limit int) ([]User, error) {
	filterQuery := []string{}
	filterArgs := map[string]interface{}{}
	for n, v := range filter {
		switch n {
		case "username":
			filterQuery = append(filterQuery, "username like %%:username%%")
			filterArgs["username"] = v
		case "account_id":
			filterQuery = append(filterQuery, "account_id=:account_id")
			filterArgs["account_id"] = v
		case "active":
			filterQuery = append(filterQuery, "active=:active")
			filterArgs["active"] = v
		case "admin":
			filterQuery = append(filterQuery, "admin=:admin")
			filterArgs["admin"] = v
		default:
			return nil, errors.Errorf("unknown filter field \"%s\"", n)
		}
	}

	query := "select id,account_id,username,admin,active,expiry from users"
	for i, f := range filterQuery {
		if i == 0 {
			query += " where " + f
		} else {
			query += " and " + f
		}
	}

	var users []User
	if err := NamedSelect(&users, query, filterArgs); err != nil {
		return nil, errors.Wrapf(err, "failed to select users")
	}
	return users, nil
}

type NewUser struct {
	Account  Account    `json:"-"` //from session
	Username string     `json:"username"`
	Password string     `json:"password"`
	Admin    bool       `json:"admin"`
	Active   bool       `json:"active"`
	Expiry   *time.Time `json:"expiry"`
}

func AddUser(nu NewUser) (*User, error) {
	if nu.Account.ID == "" {
		return nil, errors.Errorf("missing account_id")
	}
	if nu.Username == "" {
		return nil, errors.Errorf("missing username")
	}
	if nu.Password == "" {
		return nil, errors.Errorf("missing password")
	}
	stmt, err := getCompiledStatement("insert into users set id=:id,account_id=:account_id,username=:username,passhash=:passhash,admin=:admin,active=:active,expiry=:expiry")
	if err != nil {
		return nil, errors.Errorf("failed to prepare")
	}
	u := User{
		ID:       uuid.New().String(),
		Account:  nu.Account,
		Username: nu.Username,
		Active:   nu.Active,
		Admin:    nu.Admin,
		Expiry:   nu.Expiry,
	}
	passHash := passwordHash(u, nu.Password)
	if _, err := stmt.Exec(
		map[string]interface{}{
			"id":         u.ID,
			"account_id": u.Account.ID,
			"username":   u.Username,
			"passhash":   passHash,
			"admin":      u.Admin,
			"active":     u.Active,
			"expiry":     u.Expiry,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to insert user")
	}
	return &u, nil
}

func passwordHash(u User, password string) string {
	h := sha1.New()
	s := u.ID
	s += u.Account.ID
	s += password
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(req LoginRequest) (*Session, error) {
	if req.Username == "" {
		return nil, errors.Errorf("missing username")
	}
	if req.Password == "" {
		return nil, errors.Errorf("missing password")
	}

	var info struct {
		UserID        string   `db:"user_id"`
		PassHash      string   `db:"passhash"`
		UserActive    bool     `db:"user_active"`
		UserAdmin     bool     `db:"user_admin"`
		UserExpiry    *SqlTime `db:"user_expiry"`
		AccountID     string   `db:"account_id"`
		AccountName   string   `db:"account_name"`
		AccountActive bool     `db:"account_active"`
		AccountAdmin  bool     `db:"account_admin"`
		AccountExpiry *SqlTime `db:"account_expiry"`
	}
	if err := NamedGet(
		&info,
		"select u.id as user_id,u.passhash,u.active as user_active,u.admin as user_admin,u.expiry as user_expiry,"+
			"a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin,"+
			"a.expiry as account_expiry"+
			" from users as u"+
			" LEFT JOIN accounts as a on a.id = u.account_id"+
			" WHERE u.username=:username",
		map[string]interface{}{
			"username": req.Username,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to prepare")
	}
	log.Debugf("Read: %+v", info)
	passHash := passwordHash(User{ID: info.UserID, Account: Account{ID: info.AccountID}}, req.Password)
	if info.PassHash != passHash {
		log.Debugf("Login user(%s).password(%s): %s != %s", req.Username, req.Password, info.PassHash, passHash)
		return nil, errors.Errorf("wrong password")
	}
	if info.AccountExpiry != nil && time.Time(*info.AccountExpiry).Before(time.Now()) {
		return nil, errors.Errorf("account %s expired", info.AccountName)
	}
	if !info.AccountActive {
		return nil, errors.Errorf("account suspended")
	}
	if info.UserExpiry != nil && time.Time(*info.UserExpiry).Before(time.Now()) {
		return nil, errors.Errorf("user(%s) login expired", req.Username)
	}
	if !info.UserActive {
		return nil, errors.Errorf("user(%s) suspended", req.Username)
	}
	//delete existing session
	if _, err := db.NamedExec("DELETE FROM `sessions` where user_id=:user_id", map[string]interface{}{"user_id": info.UserID}); err != nil {
		return nil, errors.Wrapf(err, "failed to delete existing user session")
	}
	//create new session
	token := uuid.New().String()
	if _, err := db.NamedExec(
		"INSERT INTO `sessions` set `token`=:token,account_id=:account_id,user_id=:user_id",
		map[string]interface{}{
			"token":      token,
			"account_id": info.AccountID,
			"user_id":    info.UserID,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to create session")
	}
	return &Session{
		Token: token,
		User: User{
			ID:       info.UserID,
			Username: req.Username,
			Account: Account{
				ID:     info.AccountID,
				Name:   info.AccountName,
				Active: info.AccountActive,
				Admin:  info.AccountAdmin,
				Expiry: (*time.Time)(info.AccountExpiry),
			},
			Admin:  info.UserAdmin,
			Active: info.UserActive,
			Expiry: (*time.Time)(info.UserExpiry),
		},
	}, nil
}

func Logout(token string) error {
	if _, err := db.NamedExec(
		"DELETE FROM sessions WHERE token=:token",
		map[string]interface{}{
			"token": token,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to delete session")
	}
	return nil
}

func GetSession(token string) (*Session, error) {
	if token == "" {
		return nil, errors.Errorf("missing token")
	}
	var info struct {
		StartTime     *SqlTime `db:"time_created"`
		UserID        string   `db:"user_id"`
		Username      string   `db:"username"`
		UserActive    bool     `db:"user_active"`
		UserAdmin     bool     `db:"user_admin"`
		UserExpiry    *SqlTime `db:"user_expiry"`
		AccountID     string   `db:"account_id"`
		AccountName   string   `db:"account_name"`
		AccountActive bool     `db:"account_active"`
		AccountAdmin  bool     `db:"account_admin"`
		AccountExpiry *SqlTime `db:"account_expiry"`
	}
	if err := NamedGet(
		&info,
		"select s.time_created,s.user_id,u.username,u.active as user_active,u.admin as user_admin,u.expiry as user_expiry,"+
			"a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin,"+
			"a.expiry as account_expiry"+
			" from sessions as s"+
			" LEFT JOIN users as u on u.id = s.user_id"+
			" LEFT JOIN accounts as a on a.id = s.account_id"+
			" WHERE s.token=:token",
		map[string]interface{}{
			"token": token,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to prepare")
	}
	log.Debugf("Read: %+v", info)

	now := SqlTime(time.Now())
	if _, err := db.NamedExec(
		"UPDATE sessions SET time_updated=:now WHERE token=:token",
		map[string]interface{}{
			"now":   now,
			"token": token,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to update session")
	}
	return &Session{
		Token:       token,
		TimeCreated: info.StartTime,
		TimeUpdated: &now,
		User: User{
			ID:       info.UserID,
			Username: info.Username,
			Account: Account{
				ID:     info.AccountID,
				Name:   info.AccountName,
				Active: info.AccountActive,
				Admin:  info.AccountAdmin,
				Expiry: (*time.Time)(info.AccountExpiry),
			},
			Admin:  info.UserAdmin,
			Active: info.UserActive,
			Expiry: (*time.Time)(info.UserExpiry),
		},
	}, nil
}

func ChangePassword(userID string, newPassword string) error {
	//todo: make sure allowed to change

	if newPassword == "" {
		return errors.Errorf("missing new password")
	}
	if userID == "" {
		return errors.Errorf("missing user.id")
	}

	//read account id from user table - needed for password hashing
	var row struct {
		AccountID string `db:"account_id"`
	}
	if err := NamedGet(
		&row,
		"SELECT account_id FROM users where id=:user_id",
		map[string]interface{}{
			"user_id": userID,
		},
	); err != nil {
		return errors.Wrapf(err, "cannot read user record")
	}

	passhash := passwordHash(
		User{ID: userID, Account: Account{ID: row.AccountID}},
		newPassword,
	)
	if _, err := db.NamedExec(
		"UPDATE users SET passhash=:passhash WHERE id=:user_id",
		map[string]interface{}{
			"user_id":  userID,
			"passhash": passhash,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to change user(%s).password(%s)", userID, newPassword)
	}
	return nil
}
