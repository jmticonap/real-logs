package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func OpenDb(tagTable ...string) *sql.DB {
	var err error

	if db != nil {
		return db
	}

	db, err = sql.Open("sqlite3", "./logs.db")
	if err != nil {
		log.Fatalf("Open Db error: %s", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Ping error: %s", err)
	}

	if len(tagTable) == 0 {
		_, err = db.Exec(`
			DROP TABLE IF EXISTS performance_logs;
			CREATE TABLE IF NOT EXISTS performance_logs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				trace_id TEXT NOT NULL,
				method TEXT,
				exectime INTEGER,
				memory_mb REAL,
				timestamp DATETIME
			);
		`)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Println("Tabla creada o ya existía")
		}

		log.Println("Open successfully...")
	} else {
		query := fmt.Sprintf(`
			DROP TABLE IF EXISTS performance_logs_%s;
			CREATE TABLE IF NOT EXISTS performance_logs_%s (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				trace_id TEXT NOT NULL,
				method TEXT,
				duration_ms INTEGER,
				memory_mb REAL,
				timestamp DATETIME
			);
		`, tagTable[0], tagTable[0])
		_, err = db.Exec(query)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Println("Tabla creada o ya existía")
		}

		log.Println("Open successfully...")
	}
	return db
}
