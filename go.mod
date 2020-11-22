module github.com/acuas/sin

go 1.15

require (
	github.com/acuas/sin/db v0.0.0
	github.com/deuill/go-php v0.0.0-20181001205857-9d111e73423d // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
)

replace (
	github.com/acuas/sin/db => ./db
)