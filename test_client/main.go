package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"bitbucket.org/vservices/hotseat/db"
	"github.com/go-msvc/errors"
	"github.com/stewelarend/logger"
)

var log = logger.New().WithLevel(logger.LevelDebug)

func main() {
	addr := "http://localhost:3000"

	//get system admin user token
	adminPassword := "abc"

	//login as admin user
	var loginResp db.Session
	Post(addr+"/login", "", map[string]interface{}{"username": "admin", "password": adminPassword}, 200, &loginResp)
	log.Debugf("Admin Login: %+v", loginResp)
	t := loginResp.Token

	//create new account + account admin user
	var testAccAdminUser db.User
	var tempPassword string
	{
		type NewAccountResponse struct {
			AdminUser     db.User `json:"admin_user"`
			AdminPassword string  `json:"admin_password"`
		}
		var resp NewAccountResponse
		Post(addr+"/accounts", t, map[string]interface{}{"name": "test1"}, 200, &resp)
		log.Debugf("Added account: %+v", resp)
		testAccAdminUser = resp.AdminUser
		tempPassword = resp.AdminPassword
	}
	//admin logout
	Post(addr+"/logout", t, nil, 200, nil)

	//now work with new test account
	//first need to change admin password
	//acc admin login should fail on temp password
	Post(addr+"/login", "", map[string]interface{}{"username": testAccAdminUser.Username, "password": tempPassword}, 200, &loginResp)
	log.Debugf("ACC Admin Login: %+v", loginResp)
	t = loginResp.Token

	//acc admin create more acc users
	Post(addr+"/users", t, map[string]interface{}{"username": "one", "password": "one", "account_id": testAccAdminUser.Account.ID}, 200, nil)

	//acc admin logout
	Post(addr+"/logout", t, nil, 200, nil)

	//login as those users (force password change for each)

	//logout acc users

	//todo: delete the test account and all its users etc...

}

func Post(url string, token string, body map[string]interface{}, expCode int, respPtr interface{}) {
	bodyBuffer := bytes.NewBuffer(nil)
	if body != nil {
		json.NewEncoder(bodyBuffer).Encode(body)
	}
	httpReq, err := http.NewRequest("POST", url, bodyBuffer)
	if err != nil {
		panic(errors.Wrapf(err, "failed to make HTTP request"))
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		httpReq.Header.Set("X-Auth-Token", token)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	//resp, err := http.DefaultClient.Post(url, "application/json", bodyBuffer)
	if err != nil {
		panic(errors.Wrapf(err, "HTTP POST %s failed", url))
	}
	if resp.StatusCode != expCode {
		panic(errors.Errorf("HTTP POST %s -> %d:%s instead of %d:%s", url, resp.StatusCode, http.StatusText(resp.StatusCode), expCode, http.StatusText(expCode)))
	}
	if respPtr != nil {
		if err := json.NewDecoder(resp.Body).Decode(respPtr); err != nil {
			panic(errors.Wrapf(err, "failed to decode"))
		}
	}
	log.Debugf("HTTP POST %s -> %d %+v", url, resp.StatusCode, respPtr)
}
