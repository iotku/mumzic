package database

import (
	"database/sql"
	"github.com/iotku/mumzic/config"
	"os"
)

var SongDB *sql.DB

func init() {
	var err error
	if _, err = os.Stat(config.SongDB); os.IsNotExist(err) {
		// create new database
		return
	}
	SongDB, err = sql.Open("sqlite3", config.SongDB)
	checkErrPanic(err)
}

// Query SQLite database to count maximum amount of rows, as to not point to non-existent ID
func GetMaxID() int {
	if _, err := os.Stat(config.SongDB); os.IsNotExist(err) {
		return 0
	}

	var count int
	checkErrPanic(SongDB.QueryRow("select max(ROWID) from music;").Scan(&count))
	return count
}

// Aggressively fail on error
func checkErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func Close() {
	checkErrPanic(SongDB.Close())
}
