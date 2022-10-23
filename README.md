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

## API ##
See PostMan definition file

# Default db contents:
- system admin account
- system admin user with password "admin"

# DONE
* login/out works
* accounts can add
* users can be added by account admin
* can create and GET groups
* added list of persons and retrieve list with GET /persons
* POST /register for public users with profile
* POST /activate for public users
* POST /accounts also create user with email but no profile and requires user activation and login
* POST /groups can create the parent group
* Added meta on groups
* Busy with fields on groups

# NEXT
* After group invite was sent:
  - PUT from sub-group account id must update the group and set invitation=false and only then show in list of groups
  - Also allow DELETE if not interested by that account admin
  - Set fields
  - Set meta for cost
  - Get total cost
  - See that public users cannot see groups that are only invitations
  - Also not show groups that have sub-groups
  - Allow when group becomes active for membership
  - Membership could be module working from meta "members_xxx" only?
  - Make list of groups that user can join? Search for "Midstream" for new users, or send invite to last year's members and show related groups in browser
* Define fields on parent and child groups
  * get list of all fields to join child group
  * submit field values along with application to join the group
  * notify admin of new applications
  * review applications and accept/reject and notify the user
  * also apply for non-users
  * determine cost and take payment into each group wallet
* allow other account (by id) to create sub group (need way to repeat this next year with new group, i.e. clone)
* other account create sub group if allowed
* make form for application to join the group

* in account (Voortrekkers) create group "Lede 2022"
      init group is inactive, i.e. persons cannot join
      manage join requirements:
          cost: <amount>
          fields []   (e.g. person.name)
              can also be things not specified in person or ... but then must be populated in join request
          accept: always, manual (or later script i.e. HTTP POST ... with expected result)
      indicate join on sub-group or this group
        (Voortrekkers use sub group from a commando)
        set "is_parent_group":{...}
      for subs to be created, they need this ID and after create they will be accepted or rejected
        append account IDs allowed to create such groups
      on sub-create send notice to admin
        accept: al

* need inbox per user for notifications and to send messages between users
  create invite friend link to send with external system (whatsApp/email/...)

*     make group active to allow persons to join
*   test user join group
*
* share group ID with other accounts (e.g. Midstream Voortrekkers) (using external comms) so they can create sub-group
  (with same title) in their own account
  adding cost and adding more fields/rules/...
  when user join this group - they have to apply to both parent and child

* test family joining a group
* user wallet to pay from
* account wallet to pay into when joining group
*   compound group - pay both wallets

* let person user create more persons for own family
* let user join groups
    and make groups restrict membership e.g. with motivation and review, or with national id list check etc
    and allow existing public users to become group admins or account admins
* add wallet to user
    deposit into wallet
    let user pay for membership
* add wallet to account and sub-groups with own wallets and own admins
    may be each kommando is own account,admin,wallet but then apply to be part of group "Voortrekkers Kommandos" (i.e. list of accounts), incurring membership cost and being listed as part of it.
* add other items to pay, e.g. events, subscriptions, documents, products to be delivered
* restrict account usage and buy more range
* example of various school groups in an account
* example of various interest groups vs paid members in an account
* example of voortrekkers groups where admin of parent group determine cost and each sub group has different cost going into group wallet

BEFORE LAUNCH:
- Unit Tests!
- Restrict Use
- Load Tests
- Document



# TODO
- After create account (+acc admin user), the temp password must only be good for change password before login (at moment it works for login too, but sysadmin could have seen that password, so not safe)
- Public account + users that register themselves as member of public account - account must allow it, while other accounts require users to be created by account admin.
- User upd/del to make admin/revoke admin rights, suspend or set expiry etc...
- Delete account with all its contents (sysadmin only)
- External auth/user management for an account for list of users and login check then creaste local session
- Scripting and hooks