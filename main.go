package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"bitbucket.org/vservices/hotseat/db"
	api "bitbucket.org/vservices/hotseat/go-api"
	"github.com/go-msvc/errors"
	"github.com/gorilla/mux"
	"github.com/stewelarend/logger"
)

var log = logger.New().WithLevel(logger.LevelDebug)

func main() {
	api.New(
		map[string]map[string]api.Handler{
			"/login": {
				"POST": login, //not authed else cannot login
			},
			"/logout": {
				"POST": auth(logout),
			},
			"/users": {
				"GET":  auth(getUsers),
				"POST": auth(addUser), //add account user - must be done by account admin
			},
			"/user/{id}": {
				"GET":    auth(getUser),
				"PUT":    auth(updUser),
				"DELETE": auth(delUser),
			},
			"/user/{user_id}/password": {
				"PUT": auth(updUserPassword),
			},
			"/accounts": {
				"GET":  auth(getAccounts),
				"POST": auth(addAccount),
			},
			"/account": {
				"GET": auth(getAccount),
			},
			"/account/{account_id}": {
				"GET":    auth(getAccount),
				"PUT":    auth(updAccount),
				"DELETE": auth(delAccount),
			},
			"/groups": {
				"GET":  auth(getGroups),
				"POST": auth(addGroup),
			},
			"/group/{group_id}": {
				"GET":    auth(getGroup),
				"PUT":    auth(updGroup),
				"DELETE": auth(delGroup),
			},
			"/group/{group_id}/members": {
				"GET":  auth(getGroupMembers),
				"POST": auth(addGroupMember),
			},
			"/group/{group_id}/member/{member_id}": {
				"GET":    auth(getGroupMember),
				"PUT":    auth(updGroupMember),
				"DELETE": auth(delGroupMember),
			},
		},
	).Serve()
}

func auth(f api.ContextHandler) api.Handler {
	return func(httpRes http.ResponseWriter, httpReq *http.Request) {
		token := httpReq.Header.Get("X-Auth-Token")
		session, err := db.GetSession(token)
		if err != nil {
			log.Errorf("%+v", err)
			http.Error(httpRes, fmt.Sprintf("unauthorized: %s", err), http.StatusUnauthorized)
			return
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, db.Session{}, *session)
		log.Debugf("HTTP %s %s %+v",
			httpReq.Method,
			httpReq.URL.Path,
			session)

		status, res := f(ctx, httpRes, httpReq)
		log.Debugf("status=%d, res=(%T)%+v", status, res, res)

		//error response
		if res != nil {
			if err, ok := res.(error); ok {
				http.Error(httpRes, fmt.Sprintf("%+v", err), status)
				return
			}
		}

		//no response or non-error response
		httpRes.WriteHeader(status)
		if res != nil {
			httpRes.Header().Set("Content-Type", "application/json")
			json.NewEncoder(httpRes).Encode(res)
		}
	}
} //auth()

func login(httpRes http.ResponseWriter, httpReq *http.Request) {
	var loginRequest db.LoginRequest
	if err := json.NewDecoder(httpReq.Body).Decode(&loginRequest); err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := db.Login(loginRequest)
	if err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(*session)
}

func logout(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	if err := db.Logout(session.Token); err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to logout")
	}
	return http.StatusOK, nil
}

func getUsers(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	users, err := db.GetUsers(
		map[string]interface{}{
			"account_id": session.User.Account.ID,
		},
		[]string{"username"},
		10)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to get users")
	}
	return http.StatusOK, &users
}

func addUser(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)

	//sysadmin cannot create users - only accounts which will have account admin user
	if session.User.Account.Admin {
		return http.StatusUnauthorized, errors.Errorf("system admin cannot create account users")
	}
	if !session.User.Admin {
		return http.StatusUnauthorized, errors.Errorf("you are not account admin and therefore cannot add users to this account")
	}
	var newUser db.NewUser
	if err := json.NewDecoder(httpReq.Body).Decode(&newUser); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "failed to decode user from body")
	}

	addedUser, err := db.AddUser(
		db.NewUser{
			Account:  session.User.Account,
			Username: newUser.Username,
			Password: newUser.Password,
			Admin:    false,
			Active:   true,
			Expiry:   nil,
		},
	)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to add user")
	}
	return http.StatusOK, addedUser
}

func getUser(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func updUser(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func delUser(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func updUserPassword(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	pathVars := mux.Vars(httpReq)
	userID := pathVars["user_id"]
	newPassword := httpReq.URL.Query().Get("new_password")
	log.Debugf("Change(%s,%s)", userID, newPassword)
	if err := db.ChangePassword(userID, newPassword); err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to change password")
	}
	return http.StatusOK, nil
}

func getAccounts(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	filter := db.AccountsFilter{}
	if n := httpReq.URL.Query().Get("name"); n != "" {
		filter.Name = &n
	}
	if !session.User.Account.Admin {
		filter.ID = &session.User.Account.ID //if not sys admin: can only see own account
		falseValue := false
		filter.Admin = &falseValue //not for others to see system account
		trueValue := true
		filter.Active = &trueValue //others only see active account
	}
	accounts, err := db.GetAccounts(
		filter,
		[]string{"name"},
		urlParamInt(httpReq, "limit", 1, 100, 10))
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to get accounts")
	}
	return http.StatusOK, accounts
}

//add a new account which returns the account admin user with the account details and the admin
//user password which should be changed on first login
//(todo: password must be marked as expired in user account to force reset before login)
func addAccount(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	if !session.User.Admin || !session.User.Account.Admin {
		return http.StatusUnauthorized, errors.Errorf("accounts can only be created by system admin users")
	}
	var newAccount db.NewAccount
	if err := json.NewDecoder(httpReq.Body).Decode(&newAccount); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "failed to decode body")
	}
	accountAdminUser, adminPassword, err := db.AddAccount(newAccount)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to add account")
	}
	type NewAccountResponse struct {
		AdminUser     db.User `json:"admin_user"`
		AdminPassword string  `json:"admin_password"`
	}
	return http.StatusOK, NewAccountResponse{
		AdminUser:     *accountAdminUser,
		AdminPassword: adminPassword,
	}
}

func getAccount(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	log.Debugf("HTTP %s %s %+v",
		httpReq.Method,
		httpReq.URL.Path,
		session)

	accountID := mux.Vars(httpReq)["account_id"]
	if accountID == "" {
		accountID = session.User.Account.ID //default to own account if not specified
	}

	if !session.User.Account.Admin {
		if accountID != session.User.Account.ID {
			return http.StatusUnauthorized, errors.Errorf("not your account")
		}
	}

	log.Debugf("getAccount(%s)", accountID)
	account, err := db.GetAccount(accountID)
	if err != nil {
		log.Errorf("getAccount(%s): %+v", accountID, err)
		return http.StatusNotFound, errors.Wrapf(err, "account not found")
	}
	return http.StatusOK, account
}

func updAccount(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func delAccount(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func getGroups(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	filter := db.GroupsFilter{}

	if session.User.Account.Admin {
		//sysadmin
		if aid := httpReq.URL.Query().Get("account_id"); aid != "" {
			filter.AccountID = &aid
		}
		if ot := httpReq.URL.Query().Get("owner_type"); ot != "" {
			filter.OwnerType = &ot
		}
		if oid := httpReq.URL.Query().Get("owner_id"); oid != "" {
			filter.OwnerID = &oid
		}
	} else {
		//not sysadmin: see only own account
		filter.AccountID = &session.User.Account.ID
		//todo: see groups owned by own account or own user
	}

	if n := httpReq.URL.Query().Get("name"); n != "" {
		filter.Name = &n
	}

	userGroups, err := db.GetGroups(
		filter,
		[]string{"name"},
		urlParamInt(httpReq, "limit", 1, 100, 10))
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to get groups")
	}
	return http.StatusOK, userGroups
} //getGroups()

func addGroup(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	var newGroup struct {
		Name         string `json:"name"`
		AccountGroup bool   `json:"account_group"` //set this to true to make group owned by your account (only if you are account admin user!)
		//todo: later also allow create of group with other types of owners... and delete when owner is deleted
	}
	if err := json.NewDecoder(httpReq.Body).Decode(&newGroup); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "failed to decode body")
	}

	groupSpec := db.Group{
		Account: &session.User.Account,
		Name:    newGroup.Name,
	}
	if newGroup.AccountGroup {
		if !session.User.Admin {
			return http.StatusUnauthorized, errors.Errorf("account group can only be created by account admin users")
		}
		groupSpec.OwnerType = "account"
		groupSpec.OwnerID = session.User.Account.ID
	} else {
		groupSpec.OwnerType = "user"
		groupSpec.OwnerID = session.User.ID
	}

	g, err := db.AddGroup(groupSpec)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to add group")
	}
	return http.StatusOK, g
}

func getGroup(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	groupID := mux.Vars(httpReq)["group_id"]
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("missing /group/{group_id}")
	}

	log.Debugf("getGroup(%s)", groupID)
	g, err := db.GetGroup(groupID)
	if err != nil {
		log.Errorf("getGroup(%s): %+v", groupID, err)
		return http.StatusNotFound, errors.Wrapf(err, "group not found")
	}
	return http.StatusOK, g
} //getGroup()

func updGroup(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	// session := ctx.Value(db.Session{}).(db.Session)
	// groupID := mux.Vars(httpReq)["group_id"]
	// if groupID == "" {
	// 	return http.StatusBadRequest, errors.Errorf("missing /group/{group_id}")
	// }
	// var g model.Group
	// if err := json.NewDecoder(httpReq.Body).Decode(&g); err != nil {
	// 	return http.StatusBadRequest, errors.Wrapf(err, "cannot decode request")
	// }
	// if g.ID == "" {
	// 	g.ID = groupID
	// } else if g.ID != groupID {
	// 	return http.StatusBadRequest, errors.Errorf("group id in URL and body mismatch")
	// }

	// updatedGroup, err := db.UpdGroup(session.User, g)
	// if err != nil {
	// 	return http.StatusMethodNotAllowed, errors.Wrapf(err, "group not updated")
	// }
	// return http.StatusOK, updatedGroup
	return http.StatusInternalServerError, errors.Errorf("NYI")
} //updGroup()

func delGroup(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	// session := ctx.Value(db.Session{}).(db.Session)
	// groupID := mux.Vars(httpReq)["group_id"]
	// if groupID == "" {
	// 	return http.StatusBadRequest, errors.Errorf("missing /group/{group_id}")
	// }
	// if err := db.DelGroup(session.User, groupID); err != nil {
	// 	return http.StatusMethodNotAllowed, errors.Wrapf(err, "group not deleted")
	// }
	// return http.StatusNoContent, nil
	return http.StatusInternalServerError, errors.Errorf("NYI")
} //delGroup()

func getGroupMembers(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)

	//get group info
	group, err := db.GetGroup(mux.Vars(httpReq)["group_id"])
	if err != nil {
		return http.StatusNotFound, errors.Errorf("group(%s) not found", mux.Vars(httpReq)["group_id"])
	}

	//can only see your own group members
	switch group.OwnerType {
	case "user":
		if group.OwnerID != session.User.ID {
			return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to another user - you cannot see the members", group.ID)
		}
	case "account":
		if group.OwnerID != session.User.Account.ID {
			return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to another account - you cannot see the members", group.ID)
		}
	default:
		return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to a %s - you cannot see the members", group.ID, group.OwnerType)
	}
	members, err := db.GetGroupMembers(
		group.ID,
		db.GroupMembersFilter{},
		nil,
		urlParamInt(httpReq, "limit", 1, 100, 10)) //todo filter and sort
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to get members")
	}
	return http.StatusOK, members
} //getGroupMembers()

func addGroupMember(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)

	//get group info
	group, err := db.GetGroup(mux.Vars(httpReq)["group_id"])
	if err != nil {
		return http.StatusNotFound, errors.Errorf("group(%s) not found", mux.Vars(httpReq)["group_id"])
	}

	//can only manage your own group members
	switch group.OwnerType {
	case "user":
		if group.OwnerID != session.User.ID {
			return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to another user - you cannot manage the members", group.ID)
		}
	case "account":
		if group.OwnerID != session.User.Account.ID {
			return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to another account - you cannot manage the members", group.ID)
		}
		if !session.User.Admin {
			return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to your account - only account admin can manage the members", group.ID)
		}
	default:
		return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to a %s - you cannot manage the members", group.ID, group.OwnerType)
	}

	//see what is being added
	var newMember struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.NewDecoder(httpReq.Body).Decode(&newMember); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "cannot decode JSON body")
	}

	members, err := db.AddGroupMember(
		group.ID,
		newMember.Type,
		newMember.ID)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to add as group member")
	}
	return http.StatusOK, members
}

func getGroupMember(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	// session := ctx.Value(db.Session{}).(db.Session)
	// groupID := mux.Vars(httpReq)["group_id"]
	// groupID := mux.Vars(httpReq)["member_id"]
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func updGroupMember(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	// session := ctx.Value(db.Session{}).(db.Session)
	// groupID := mux.Vars(httpReq)["group_id"]
	// groupID := mux.Vars(httpReq)["member_id"]
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func delGroupMember(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	// session := ctx.Value(db.Session{}).(db.Session)
	// groupID := mux.Vars(httpReq)["group_id"]
	// groupID := mux.Vars(httpReq)["member_id"]
	return http.StatusInternalServerError, errors.Errorf("NYI")
}

func urlParamInt(httpReq *http.Request, paramName string, min, max, def int) int {
	i := 10
	if s := httpReq.URL.Query().Get(paramName); s != "" {
		if i64, err := strconv.ParseInt(s, 10, 64); err == nil {
			i = int(i64)
			if i < min {
				i = min
			}
			if i > max {
				i = max
			}
		}
	}
	return i
}
