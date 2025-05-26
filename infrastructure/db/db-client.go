package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/jmticonap/real-logs/domain"
	_ "github.com/mattn/go-sqlite3"
)

var db = map[string]*sql.DB{}
var dbMutex sync.Mutex

func OpenDb(params domain.StrObject) *sql.DB {
	var err error
	var dir string
	var dbPath string
	var dirOk bool

	dbMutex.Lock()
	if dir, dirOk = params["dir"]; dirOk {
		os.Mkdir(dir, DirGrants(true, true, true))
		dbPath = filepath.Join(dir, "log.db")
	} else {
		dir = "."
		dbPath = "./log.db"
	}

	if _, dbOk := db[dir]; dbOk {
		dbMutex.Unlock()
		return db[dir]
	}

	db[dir], err = sql.Open("sqlite3", dbPath)
	if err != nil {
		dbMutex.Unlock()
		log.Fatalf("Open Db error: %s", err)
	}

	err = db[dir].Ping()
	if err != nil {
		dbMutex.Unlock()
		log.Fatalf("Ping error: %s", err)
	}

	_, err = db[dir].Exec(`
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
		dbMutex.Unlock()
		log.Fatal(err)
	} else {
		log.Println("Tabla creada o ya existía [performance_logs]")
	}

	_, err = db[dir].Exec(`
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
		dbMutex.Unlock()
		log.Fatal(err)
	} else {
		log.Println("Tabla creada o ya existía [general_logs]")
	}

	log.Println("Open successfully...")

	dbMutex.Unlock()
	return db[dir]
}

func DirGrants(leer bool, escribir bool, ejecutar bool) os.FileMode {
	perm := 0
	if leer {
		perm |= 4
	}
	if escribir {
		perm |= 2
	}
	if ejecutar {
		perm |= 1
	}
	// Aplicar los permisos para el propietario, grupo y otros de forma idéntica
	// para este ejemplo simple. Puedes ajustarlo si necesitas más granularidad.
	fileMode := os.FileMode(perm<<6 | perm<<3 | perm)
	// Añadir el bit de directorio
	return fileMode | os.ModeDir
}
