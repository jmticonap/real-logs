package db

import (
	"database/sql"
	"log"
	"path/filepath"

	"github.com/jmticonap/real-logs/domain"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func OpenDb(params domain.StrObject) *sql.DB {
	var err error

	if db != nil {
		return db
	}

	var dbPath string
	if dir, dirOk := params["dir"]; dirOk {
		dbPath = filepath.Join(dir, "log.db")
	} else {
		dbPath = "./log.db"
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Open Db error: %s", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Ping error: %s", err)
	}

	_, err = db.Exec(`
		DROP TABLE IF EXISTS performance_logs;
		CREATE TABLE IF NOT EXISTS performance_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT NOT NULL,
			method TEXT,
			exectime INTEGER,
			memory_mb VARCHAR(10),
			timestamp DATETIME
		);
	`)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Tabla creada o ya existía [performance_logs]")
	}

	_, err = db.Exec(`
		DROP TABLE IF EXISTS general_logs;
		CREATE TABLE IF NOT EXISTS general_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level CHARACTER(15),
			timestamp DATETIME,
			pid NUMERIC,
			hostname VARCHAR(255),
			trace_id VARCHAR(40),
			span_id VARCHAR(40),
			parent_id VARCHAR(40),
			msg TEXT
		);
	`)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Tabla creada o ya existía [general_logs]")
	}

	log.Println("Open successfully...")

	return db
}
