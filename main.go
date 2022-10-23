package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
			"/register": {
				"POST": register,
			},
			"/activate": {
				"POST": activate,
			},
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
			"/user/{user_id}": {
				"GET":    auth(getUser),
				"PUT":    auth(updUser),
				"DELETE": auth(delUser),
			},
			"/user/{user_id}/password": {
				"PUT": auth(updUserPassword),
			},
			"/messages": {
				"POST": auth(sendMessage),
				"GET":  auth(getMessages),
			},
			"/message/{message_id}": {
				"GET": auth(getMessage),
				"PUT": auth(updMessage),
				"DEL": auth(delMessage),
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
				"GET":  auth(getGroups, "Get list of groups owned by your account as well as groups that your account are allowed to create a sub-group in, even if you already did so."),
				"POST": auth(addGroup, "Create a new group that belongs to the account. Only account admin can create a group."),
			},
			"/group/{group_id}": {
				"GET":    auth(getGroup),
				"PUT":    auth(updGroup),
				"DELETE": auth(delGroup),
			},
			"/group/{group_id}/fields": {
				"GET":    auth(getGroupFields),
				"PUT":    auth(updGroupFields),
				"DELETE": auth(delGroupFields),
			},
			"/persons": {
				"GET": auth(getPersons),
			},
		},
	).Serve()
}

func auth(f api.ContextHandler, args ...interface{}) api.Handler {
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
		if res != nil {
			httpRes.Header().Set("Content-Type", "application/json")
		}
		httpRes.WriteHeader(status)
		if res != nil {
			json.NewEncoder(httpRes).Encode(res)
		}
	}
} //auth()

func register(httpRes http.ResponseWriter, httpReq *http.Request) {
	var registerRequest db.RegisterRequest
	if err := json.NewDecoder(httpReq.Body).Decode(&registerRequest); err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := db.Register(registerRequest)
	if err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(*session)
}

//GET /activate?token=<token>
func activate(httpRes http.ResponseWriter, httpReq *http.Request) {
	var activateRequest db.ActivateRequest
	if err := json.NewDecoder(httpReq.Body).Decode(&activateRequest); err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := db.ActivateUser(activateRequest)
	if err != nil {
		http.Error(httpRes, err.Error(), http.StatusInternalServerError)
		return
	}
	httpRes.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpRes).Encode(*session)
}

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
	filter := map[string]interface{}{}
	if session.User.Account.Admin {
		//only filter on account_id if specified in params
		if param_account_id := httpReq.URL.Query().Get("account_id"); param_account_id != "" {
			filter["account_id"] = param_account_id
		}
	} else {
		filter["account_id"] = session.User.Account.ID
	}
	users, err := db.GetUsers(
		filter,
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
	session := ctx.Value(db.Session{}).(db.Session)
	var accountID string
	if !session.User.Account.Admin {
		accountID = session.User.Account.ID
	}
	vars := mux.Vars(httpReq)
	users, err := db.GetUser(accountID, vars["user_id"])
	if err != nil {
		return http.StatusNotFound, errors.Wrapf(err, "failed to get user")
	}
	return http.StatusOK, &users
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

func sendMessage(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)

	toUserID := httpReq.URL.Query().Get("to_user_id")
	if toUserID == "" {
		return http.StatusBadRequest, errors.Errorf("missing URL parameter to_user_id")
	}

	toUser, err := db.GetUser("" /*session.User.Account.ID*/, toUserID)
	if err != nil {
		return http.StatusBadRequest, errors.Errorf("unknown to_user_id")
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(httpReq.Body).Decode(&body); err != nil || body.Message == "" {
		return http.StatusBadRequest, errors.Errorf("missing message in body")
	}

	id, err := session.User.SendMessage(
		toUser,
		body.Message,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to send")
	}
	return http.StatusOK, struct {
		MessageID string `json:"message_id"`
	}{
		MessageID: id,
	}
}

func getMessages(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	messages, err := session.User.Inbox(httpReq.URL.Query().Get("status"), 30)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to read messages")
	}
	return http.StatusOK, messages
}

func getMessage(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	return http.StatusNotFound, errors.Errorf("NYI")
}

func updMessage(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	return http.StatusNotFound, errors.Errorf("NYI")
}

func delMessage(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	return http.StatusNotFound, errors.Errorf("NYI")
}

func getAccounts(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	filter := db.AccountsFilter{}
	if n := httpReq.URL.Query().Get("name"); n != "" {
		filter.Name = &n
		log.Debugf("filter.Name=%s", *filter.Name)
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
	} else {
		//not sysadmin: see only own account
		filter.AccountID = &session.User.Account.ID
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

	//todo: check if account is allowed to create more groups (limit nr of groups)
	if !session.User.Admin {
		return http.StatusUnauthorized, errors.Errorf("only account admin user can create groups")
	}

	var newGroup db.NewGroup
	if err := json.NewDecoder(httpReq.Body).Decode(&newGroup); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "failed to decode body")
	}
	g, err := db.AddGroup(session.User, newGroup)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to add group")
	}
	return http.StatusOK, g
}

func getGroup(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	groupID := strings.TrimSpace(mux.Vars(httpReq)["group_id"])
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("expecting /group/<group_id> in URL")
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
	session := ctx.Value(db.Session{}).(db.Session)
	groupID := strings.TrimSpace(mux.Vars(httpReq)["group_id"])
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("expecting /group/<group_id> in URL")
	}
	if !session.User.Admin {
		return http.StatusUnauthorized, errors.Errorf("only account admin user can change groups")
	}
	group, err := db.GetGroup(groupID)
	if err != nil {
		return http.StatusNotFound, nil
	}
	if group.Account.ID != session.User.Account.ID {
		return http.StatusUnauthorized, errors.Errorf("group does not belong to your account")
	}

	var changes db.Group
	if err := json.NewDecoder(httpReq.Body).Decode(&changes); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "cannot decode request")
	}

	nrChanges := 0
	if group.Name != changes.Name {
		nrChanges++
		group.Name = changes.Name
	}
	if changes.Description != nil &&
		*changes.Description != "" &&
		(group.Description == nil || *changes.Description != *group.Description) {
		nrChanges++
		group.Description = changes.Description
	}

	//set only group data that must change (nil not to change any thing, nil values to delete seleted meta names)
	group.Data = changes.Data

	//apply the changes
	if err := db.UpdGroup(session.User, *group); err != nil {
		return http.StatusMethodNotAllowed, errors.Wrapf(err, "failed to update group")
	}

	//read the updates
	group, err = db.GetGroup(groupID)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to read updated group")
	}
	return http.StatusOK, group
} //updGroup()

func delGroup(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	groupID := strings.TrimSpace(mux.Vars(httpReq)["group_id"])
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("expecting /group/<group_id> in URL")
	}
	if !session.User.Admin {
		return http.StatusUnauthorized, errors.Errorf("only account admin user can delete a group")
	}
	if err := db.DelGroup(session.User, groupID); err != nil {
		return http.StatusMethodNotAllowed, errors.Wrapf(err, "group not deleted")
	}
	return http.StatusNoContent, nil
} //delGroup()

func getGroupFields(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	//session := ctx.Value(db.Session{}).(db.Session)
	groupID := strings.TrimSpace(mux.Vars(httpReq)["group_id"])
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("expecting /group/<group_id> in URL")
	}
	includeParentFields := getBoolParam(httpReq.URL.Query().Get("include_parent_fields"), false)

	log.Debugf("getGroupFields(%s)", groupID)
	gf, err := db.GetGroupFields(groupID, includeParentFields)
	if err != nil {
		log.Errorf("getGroupFields(%s): %+v", groupID, err)
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to get group fields")
	}
	return http.StatusOK, gf
} //getGroupFields()

func updGroupFields(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	groupID := strings.TrimSpace(mux.Vars(httpReq)["group_id"])
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("expecting /group/<group_id> in URL")
	}
	if !session.User.Admin {
		return http.StatusUnauthorized, errors.Errorf("only account admin user can change groups")
	}
	group, err := db.GetGroup(groupID)
	if err != nil {
		return http.StatusNotFound, nil
	}
	if group.Account.ID != session.User.Account.ID {
		return http.StatusUnauthorized, errors.Errorf("group does not belong to your account")
	}

	var changes db.Group
	if err := json.NewDecoder(httpReq.Body).Decode(&changes); err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "cannot decode request")
	}
	return http.StatusInternalServerError, errors.Errorf("NYI")

	// nrChanges := 0
	// if group.Name != changes.Name {
	// 	nrChanges++
	// 	group.Name = changes.Name
	// }
	// if changes.Description != nil &&
	// 	*changes.Description != "" &&
	// 	(group.Description == nil || *changes.Description != *group.Description) {
	// 	nrChanges++
	// 	group.Description = changes.Description
	// }

	// //set only group data that must change (nil not to change any thing, nil values to delete seleted meta names)
	// group.Data = changes.Data

	// //apply the changes
	// if err := db.UpdGroup(session.User, *group); err != nil {
	// 	return http.StatusMethodNotAllowed, errors.Wrapf(err, "failed to update group")
	// }

	// //read the updates
	// group, err = db.GetGroup(groupID)
	// if err != nil {
	// 	return http.StatusInternalServerError, errors.Wrapf(err, "failed to read updated group")
	// }
	// return http.StatusOK, group
} //updGroupFields()

func delGroupFields(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	groupID := strings.TrimSpace(mux.Vars(httpReq)["group_id"])
	if groupID == "" {
		return http.StatusBadRequest, errors.Errorf("expecting /group/<group_id> in URL")
	}
	if !session.User.Admin {
		return http.StatusUnauthorized, errors.Errorf("only account admin user can delete a group")
	}
	return http.StatusInternalServerError, errors.Errorf("NYI")

	// if err := db.DelGroupFields(session.User, groupID); err != nil {
	// 	return http.StatusMethodNotAllowed, errors.Wrapf(err, "group not deleted")
	// }
	// return http.StatusNoContent, nil
} //delGroupFields()

func getGroupMembers(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)

	//get group info
	group, err := db.GetGroup(mux.Vars(httpReq)["group_id"])
	if err != nil {
		return http.StatusNotFound, errors.Errorf("group(%s) not found", mux.Vars(httpReq)["group_id"])
	}

	//can only see your own group members
	if !session.User.Account.Admin && group.Account.ID != session.User.Account.ID {
		return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to another account - you cannot see the members", group.ID)
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
	log.Debugf("group: %+v", group)

	//can only see your own group members
	if !session.User.Account.Admin && group.Account.ID != session.User.Account.ID {
		return http.StatusUnauthorized, errors.Errorf("group(%s) belongs to another account - you cannot see the members", group.ID)
	}

	//see what is being added
	var newMember struct {
		Type string `json:"member_type"`
		ID   string `json:"member_id"`
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

func getPersons(ctx context.Context, httpRes http.ResponseWriter, httpReq *http.Request) (status int, res interface{}) {
	session := ctx.Value(db.Session{}).(db.Session)
	if !session.User.Account.Admin {
		return http.StatusUnauthorized, nil
	}

	//sysadmin only for now (other users can get access to users they know the ID of, e.g. family members or people who entered an event)
	filter := db.PersonsFilter{}
	if n := httpReq.URL.Query().Get("name"); n != "" {
		filter.Name = &n
	}
	if n := httpReq.URL.Query().Get("surname"); n != "" {
		filter.Surname = &n
	}
	persons, err := db.GetPersons(
		filter,
		[]string{"surname", "name"},
		urlParamInt(httpReq, "limit", 1, 100, 10))
	if err != nil {
		return http.StatusInternalServerError, errors.Wrapf(err, "failed to get persons")
	}
	return http.StatusOK, persons
}

func getBoolParam(v string, d bool) bool {
	if v != "" {
		if v == "true" {
			return true
		}
		if v == "false" {
			return false
		}
	}
	return d
}
