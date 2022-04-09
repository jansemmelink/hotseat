# Getting Started
Start db with `docker-compose up -d`.
Build and run the API with `go build` and `./hotseat`

## Login as system admin
The initial db has a system admin account and system admin user belonging to that account.
Login with:

```
curl -s -D /dev/stderr -XPOST 'http://localhost:3000/login' -d'{"username":"admin","password":"admin"}'
```

If login fail, see hotseat log output for the error which will show the expected passhash. Update the users table with that hash then try again, e.g.:

```
MariaDB [hotseat]> update users set passhash='8e55ed952f2463f4a75b61669f9115730087dda2';
```

When login succeeds, you get session data like this:

```
{
  "token": "c1f365dd-1d24-4d31-9ecd-076175eb873a",
  "user": {
    "id": "7ec6f8ae-b4f9-11ec-a38f-0242ac170002",
    "account": {
      "id": "76b8f2ae-b4f9-11ec-a38f-0242ac170002",
      "name": "admin",
      "active": true,
      "admin": true,
      "expiry": null
    },
    "username": "admin",
    "admin": true,
    "active": true
  }
}
```

The token value must be used in all subsequent API calls as header X-Auth-Token.
For example, get a list of accounts with:

```
% curl -s -D /dev/stderr  -XGET 'http://localhost:3000/accounts' -HX-Auth-Token:c1f365dd-1d24-4d31-9ecd-076175eb873a | jq
HTTP/1.1 200 OK
Date: Tue, 05 Apr 2022 16:13:39 GMT
Content-Length: 192
Content-Type: text/plain; charset=utf-8

[
  {
    "id": "76b8f2ae-b4f9-11ec-a38f-0242ac170002",
    "name": "admin",
    "active": true,
    "admin": true,
    "expiry": null
  }
]
```

## Accounts ##
Login as sysadmin to create a new account with a name. It will automatically create the account admin users:

```
curl -s -D /dev/stderr  -XPOST 'http://localhost:3000/accounts' -HX-Auth-Token:c1f365dd-1d24-4d31-9ecd-076175eb873a -d'{"name":"test2"}'
HTTP/1.1 200 OK
Date: Tue, 05 Apr 2022 16:15:32 GMT
Content-Length: 242
Content-Type: text/plain; charset=utf-8

{
  "admin_user": {
    "id": "ba3e192b-5875-43fd-a328-f2eedc20ab88",
    "account": {
      "id": "fab22945-4e92-48ac-aada-e4239729dd19",
      "name": "test2",
      "active": true,
      "expiry": null
    },
    "username": "test2.admin",
    "admin": true,
    "active": true
  },
  "admin_password": "s^gL{;4nXc"
}```

Note the admin password down, as you will not see it again after this response. It is hashed in the db. You can change it manually in the db by trying to login with the password you want to use, then copy that hash from the hotseat log and update the user's passhash in the db and login again.

Login as account admin using that username and password:

```
curl -s -D /dev/stderr -XPOST 'http://localhost:3000/login' -d'{"username":"test2.admin","password":"s^gL{;4nXc"}' | jq
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 05 Apr 2022 16:18:27 GMT
Content-Length: 253

{
  "token": "a8b67121-c8ea-4414-b163-d27929182243",
  "user": {
    "id": "ba3e192b-5875-43fd-a328-f2eedc20ab88",
    "account": {
      "id": "fab22945-4e92-48ac-aada-e4239729dd19",
      "name": "test2",
      "active": true,
      "expiry": null
    },
    "username": "test2.admin",
    "admin": true,
    "active": true
  }
}
```

Use above token for subsequent API calls in this session.

Each time you login, the previous user session is deleted. It is also deleted when you logout.
Currently sessions never expire. TODO.

## Groups ##

Create a group for a user or an account with:

```
curl -s -D /dev/stdout -XPOST 'http://localhost:3000/groups' -HX-Auth-Token:a8b67121-c8ea-4414-b163-d27929182243 -d'{"name":"g1"}' | jq
HTTP/1.1 200 OK
Date: Tue, 05 Apr 2022 16:20:31 GMT
Content-Length: 227
Content-Type: text/plain; charset=utf-8

{
  "id": "339c364d-9d79-4118-b645-5b41b378da68",
  "account": {
    "id": "fab22945-4e92-48ac-aada-e4239729dd19",
    "name": "test2",
    "active": true,
    "expiry": null
  },
  "owner_type": "user",
  "owner_id": "ba3e192b-5875-43fd-a328-f2eedc20ab88",
  "name": "g1"
}
```

Get group details:

```
curl -s -D /dev/stderr -XGET 'http://localhost:3000/group/339c364d-9d79-4118-b645-5b41b378da68' -HX-Auth-Token:a8b67121-c8ea-4414-b163-d27929182243 | jq
HTTP/1.1 200 OK
Date: Tue, 05 Apr 2022 16:31:28 GMT
Content-Length: 227
Content-Type: text/plain; charset=utf-8

{
  "id": "339c364d-9d79-4118-b645-5b41b378da68",
  "account": {
    "id": "fab22945-4e92-48ac-aada-e4239729dd19",
    "name": "test2",
    "active": true,
    "expiry": null
  },
  "owner_type": "user",
  "owner_id": "ba3e192b-5875-43fd-a328-f2eedc20ab88",
  "name": "g1"
}
```


Add members to the group with:
```
% curl -s -D /dev/stdout -XPOST 'http://localhost:3000/group/339c364d-9d79-4118-b645-5b41b378da68/members' -HX-Auth-Token:a8b67121-c8ea-4414-b163-d27929182243 -d'{"member_type":"user","member_id":"...user id..."}'
```



# Default db contents:
- system admin account
- system admin user with password "admin"


# DONE
* login/out works
* accounts can add
* users can be added by account admin
* can create and GET groups

# NEXT
* Add/remove members to/from group

# TODO
- After create account (+acc admin user), the temp password must only be good for change password before login (at moment it works for login too, but sysadmin could have seen that password, so not safe)
- Public account + users that register themselves as member of public account - account must allow it, while other accounts require users to be created by account admin.
- User upd/del to make admin/revoke admin rights, suspend or set expiry etc...
- Delete account with all its contents (sysadmin only)
- External auth/user management for an account for list of users and login check then creaste local session
