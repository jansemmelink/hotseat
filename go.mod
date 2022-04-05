module bitbucket.org/vservices/hotseat

go 1.17

replace bitbucket.org/vservices/go-api => ../go-api

require (
	bitbucket.org/vservices/go-api v0.0.0-00010101000000-000000000000
	github.com/gchaincl/sqlhooks v1.3.0
	github.com/go-msvc/errors v0.0.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/google/uuid v1.3.0
	github.com/jmoiron/sqlx v1.3.4
	github.com/stewelarend/logger v0.0.4
)

require github.com/gorilla/mux v1.8.0

require github.com/go-msvc/logger v0.0.0-20210121062433-1f3922644bec // indirect
