package util

import (
	"database/sql"
	"fmt"

	"github.com/ksw2000/catch_cat_server/config"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func OpenDB() *sql.DB {
	if db != nil {
		return db
	}
	var err error
	if db, err = sql.Open("sqlite3", config.MainDB); err != nil {
		panic(fmt.Sprintf("can not connect to database: %s", config.MainDB))
	}
	return db
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}
