package database

import (
	"database/sql"
	_ "github.com/mattn/go-oci8"
)

func NewConnection() *sql.DB {
	user, _ := OracleUser()
	password, _ := OraclePassword()
	host, _ := OracleHost()
	port, _ := OraclePort()
	service, _ := OracleService()
	connectionString := user + "/" + password + "@" + host + ":" + port + "/" + service
	//log.Println(connectionString)
	db, err := sql.Open("oci8", connectionString)
	if err != nil {

		panic("Unable to create database connection")
	}

	if err = db.Ping(); err != nil {
		db.Close()
		panic("Error connecting to the database: %s\n" + err.Error())

	}
	return db
}
