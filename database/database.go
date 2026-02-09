package database

import (
	"database/sql"
	"fmt"
	"os"
)

// MediaDBPath is a database generated from genMusicSQLiteDB
var MediaDBPath = "./media.db"
var MediaDB *sql.DB

func init() {
	var err error
	if _, err = os.Stat(MediaDBPath); os.IsNotExist(err) {
		// create new database
		return
	}
	MediaDB, err = sql.Open("sqlite3", MediaDBPath)
	checkErrPanic(err)
}

// openDB returns an opened sqlite3 database
func openDB(DatabasePath string) *sql.DB {
	database, err := sql.Open("sqlite3", DatabasePath)
	checkErrPanic(err)
	return database
}

// Query SQLite database to count maximum amount of rows, as to not point to non-existent ID
func GetMaxID() int {
	if _, err := os.Stat(MediaDBPath); os.IsNotExist(err) {
		return 0
	}

	var count int
	checkErrPanic(MediaDB.QueryRow("select max(ROWID) from music;").Scan(&count))
	return count
}

// Aggressively fail on error
func checkErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func Close(database *sql.DB) {
	if database != nil {
		checkErrPanic(database.Close())
	} else {
		fmt.Println("Tried closing nil DB")
	}
}
