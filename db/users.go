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
	Account  *Account   `json:"account,omitempty" doc:"Account is nil for non-account users registering on the system"`
	Username string     `json:"username"`
	Admin    bool       `json:"admin,omitempty"`
	Active   bool       `json:"active"`
	Expiry   *time.Time `json:"expiry,omitempty"`
	Person   *Person    `json:"person,omitempty" doc:"Person linked to this user account"`
}

type userRow struct {
	UserID        string   `db:"user_id"`
	Username      string   `db:"username"`
	PassHash      string   `db:"passhash"`
	UserActive    bool     `db:"user_active"`
	UserAdmin     bool     `db:"user_admin"`
	UserExpiry    *SqlTime `db:"user_expiry"`
	AccountID     string   `db:"account_id"`
	AccountName   string   `db:"account_name"`
	AccountActive bool     `db:"account_active"`
	AccountAdmin  bool     `db:"account_admin"`
	AccountExpiry *SqlTime `db:"account_expiry"`
	PersonID      *string  `db:"person_id"`
}

func (ur userRow) User() User {
	u := User{
		ID:       ur.UserID,
		Account:  nil,
		Username: ur.Username,
		Admin:    ur.UserAdmin,
		Active:   ur.UserActive,
		Expiry:   (*time.Time)(ur.UserExpiry),
		Person:   nil,
	}
	if ur.AccountID != "" {
		u.Account = &Account{
			ID:     ur.AccountID,
			Name:   ur.AccountName,
			Active: ur.AccountActive,
			Admin:  ur.AccountAdmin,
			Expiry: (*time.Time)(ur.AccountExpiry),
		}
	}
	return u
}

const userRowQuery = "select u.id as user_id,u.username,u.passhash,u.active as user_active,u.admin as user_admin,u.expiry as user_expiry," +
	"a.id as account_id,a.name as account_name,a.active as account_active,a.admin as account_admin," +
	"a.expiry as account_expiry,u.person_id" +
	" from users as u" +
	" LEFT JOIN accounts as a on a.id = u.account_id"

func GetUsers(filter map[string]interface{}, sort []string, limit int) ([]User, error) {
	log.Debugf("GetUsers(filter:%+v, sort:%+v, limit:%v)", filter, sort, limit)
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
			filterQuery = append(filterQuery, "u.admin=:admin")
			filterArgs["admin"] = v
		default:
			return nil, errors.Errorf("unknown filter field \"%s\"", n)
		}
	}

	query := userRowQuery
	for i, f := range filterQuery {
		if i == 0 {
			query += " where " + f
		} else {
			query += " and " + f
		}
	}

	var userRows []userRow
	if err := NamedSelect(&userRows, query, filterArgs); err != nil {
		return nil, errors.Wrapf(err, "failed to select users")
	}

	users := make([]User, len(userRows))
	for i, ur := range userRows {
		users[i] = ur.User()
	}
	return users, nil
} //GetUsers()

//accountID may only be "" when called from system admin user, else the account id of the user calling this function
func GetUser(accountID string, userID string) (*User, error) {
	log.Debugf("GetUser(accountID:%s,userID:%s)", accountID, userID)
	filterQuery := []string{}
	filterArgs := map[string]interface{}{}
	if accountID != "" {
		filterQuery = append(filterQuery, "account_id=:account_id")
		filterArgs["account_id"] = accountID
	}
	filterQuery = append(filterQuery, "u.id=:user_id")
	filterArgs["user_id"] = userID

	query := userRowQuery
	for i, f := range filterQuery {
		if i == 0 {
			query += " where " + f
		} else {
			query += " and " + f
		}
	}

	var userRow userRow
	if err := NamedGet(&userRow, query, filterArgs); err != nil {
		return nil, errors.Wrapf(err, "failed to get user")
	}
	u := userRow.User()

	if userRow.PersonID != nil {
		person, err := GetPerson(*userRow.PersonID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get user.person")
		}
		u.Person = person
	}

	return &u, nil
} //GetUser()

type NewUser struct {
	Account  *Account   `json:"-"` //from session, nil for public user registration
	Username string     `json:"username"`
	Password string     `json:"password"`
	Admin    bool       `json:"admin"`
	Active   bool       `json:"active"`
	Expiry   *time.Time `json:"expiry"`
	Person   *Person    `json:"person"`
}

func AddUser(nu NewUser) (*User, error) {

	if nu.Account == nil || nu.Account.ID == "" {
		return nil, errors.Errorf("missing account_id")
	}
	if nu.Username == "" {
		return nil, errors.Errorf("missing username")
	}
	if nu.Password == "" {
		return nil, errors.Errorf("missing password")
	}
	u := User{
		ID:       uuid.New().String(),
		Account:  nu.Account,
		Username: nu.Username,
		Active:   nu.Active,
		Admin:    nu.Admin,
		Expiry:   nu.Expiry,
		Person:   nu.Person,
	}
	personValues := map[string]interface{}{
		"id":         u.ID,
		"account_id": u.Account.ID,
		"username":   u.Username,
		"passhash":   passwordHash(u, nu.Password),
		"admin":      u.Admin,
		"active":     u.Active,
		"expiry":     u.Expiry,
		"person_id":  nil,
	}
	if u.Person != nil && u.Person.ID != "" {
		personValues["person_id"] = u.Person.ID
	}
	if _, err := db.NamedExec(
		"insert into users set id=:id,account_id=:account_id,username=:username,passhash=:passhash,admin=:admin,active=:active,expiry=:expiry,person_id=:person_id",
		personValues,
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

type RegisterRequest struct {
	Email      string    `json:"email" doc:"Email is required to contact the user and to login."`
	Phone      string    `json:"phone" doc:"Phone number where person may be contacted."`
	Name       string    `json:"name" doc:"First name of the person registering as a user."`
	Surname    string    `json:"surname" doc:"Surname of the person registering as a user."`
	Dob        SqlDate   `json:"dob" doc:"Date of birth is required"`
	Gender     SqlGender `json:"gender" doc:"Gender male|female is required"`
	CountryID  string    `json:"country_id" doc:"Nationality of the person registering is indicated by a country_id"`
	NationalID string    `json:"national_id" doc:"National ID number in the above country."`
}

type RegisterResponse struct {
	Email string `json:"email" doc:"Use this to activate the account"`
	Token string `json:"token" doc:"Use this to activate the account"`
}

var registerDobMin time.Time

func init() {
	registerDobMin, _ = time.ParseInLocation("2006-01-02", "1900-01-01", time.UTC)
}

func (req RegisterRequest) Validate() error {
	if req.Name == "" {
		return errors.Errorf("missing name")
	}
	if req.Surname == "" {
		return errors.Errorf("missing surname")
	}
	if req.Gender != SqlGenderMale && req.Gender != SqlGenderFemale {
		return errors.Errorf("gender not specified")
	}
	if req.Email == "" {
		return errors.Errorf("missing email")
	}
	if req.Phone == "" {
		return errors.Errorf("missing phone")
	}
	if time.Time(req.Dob).Before(registerDobMin) || time.Time(req.Dob).After(time.Now()) {
		return errors.Errorf("dob:\"%s\" is outside range %s to current time (UTC).", time.Time(req.Dob).UTC().Format("2006-01-02"), registerDobMin)
	}

	if req.CountryID == "" {
		return errors.Errorf("missing country_id")
	}
	if req.NationalID == "" {
		return errors.Errorf("missing national_id")
	}
	return nil
}

//called to register new user from web site
func Register(req RegisterRequest) (*RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid request")
	}
	country, err := GetCountryByID(req.CountryID)
	if err != nil {
		log.Errorf("GetCountryByID failed: %+v", err)
		country, err = GetCountryByName(req.CountryID)
		if err != nil {
			log.Errorf("GetCountryByName failed: %+v", err)
			return nil, errors.Errorf("unknown country(%s)", req.CountryID)
		}
	}

	person, err := AddPersonIfNotExist(Person{
		Name:    req.Name,
		Surname: req.Surname,
		Email:   &req.Email,
		Phone:   &req.Phone,
		Dob:     &req.Dob,
		Gender:  &req.Gender,
		Nationalities: []Nationality{{
			Country:    country,
			NationalID: req.NationalID,
		}},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create personal info record")
	}

	newPassword := newRandomPassword(10)
	user, err := AddUser(NewUser{
		Account:  publicAccount,
		Username: req.Email,
		Password: newPassword,
		Admin:    false,
		Active:   false, //need to activate using current password and new password
		Expiry:   nil,
		Person:   person,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create user account")
	}

	return &RegisterResponse{
		Email: req.Email,
		Token: passwordHash(*user, newPassword),
	}, nil
}

type ActivateRequest struct {
	Email       string `json:"email" doc:"This is the email used to register"`
	Token       string `json:"token" doc:"This is the token returned from register response"` //==random password given to inactive user account
	NewPassword string `json:"new_password" doc:"New password selected by the user"`
}

func ActivateUser(req ActivateRequest) (*User, error) {
	//todo: check strength of new password
	log.Debugf("activate %+v", req)
	if req.NewPassword == "" {
		return nil, errors.Errorf("missing new_password")
	}
	if err := CheckPasswordStrength(req.NewPassword, 8); err != nil {
		return nil, errors.Wrapf(err, "new_password not strong enough")
	}

	var userRow userRow
	if err := NamedGet(&userRow, userRowQuery+" where u.username=:email and u.passhash=:token and u.active=false", map[string]interface{}{
		"email": req.Email,
		"token": req.Token,
	}); err != nil {
		return nil, errors.Wrapf(err, "user not found")
	}
	log.Debugf("activate userRow: %+v", userRow)

	user := userRow.User()
	log.Debugf("activate user: %+v", user)

	//activate the user and set new password
	if _, err := db.NamedExec("update users set active=true,passhash=:passhash where id=:id", map[string]interface{}{
		"passhash": passwordHash(user, req.NewPassword),
		"id":       user.ID,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to activate user account")
	}
	log.Debugf("user(%s) activated", user.ID)
	user.Active = true
	return &user, nil
}

//AddFamilyMember is called by a logged in parent to add a child or spouse or own parent
func AddFamilyMember() error {
	return errors.Errorf("NYI")
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

	var info userRow
	if err := NamedGet(
		&info,
		userRowQuery+
			" WHERE u.username=:username",
		map[string]interface{}{
			"username": req.Username,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to prepare")
	}
	log.Debugf("Read: %+v", info)
	passHash := passwordHash(User{ID: info.UserID, Account: &Account{ID: info.AccountID}}, req.Password)
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
		User:  info.User(),
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
		return nil, errors.Wrapf(err, "failed to get users")
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
			Account: &Account{
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
		User{ID: userID, Account: &Account{ID: row.AccountID}},
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
