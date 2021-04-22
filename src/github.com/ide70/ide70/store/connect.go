package store

import (
	"database/sql"
	"fmt"
	"github.com/ide70/ide70/util/log"
	_ "github.com/lib/pq"
)

var logger = log.Logger{"store"}

type DatabaseContext struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func (dbCtx *DatabaseContext) getConnection() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		dbCtx.Host, dbCtx.Port, dbCtx.User, dbCtx.Password, dbCtx.DBName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	return db
}
