package search

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/iotku/mumzic/config"
	_ "github.com/mattn/go-sqlite3"
)

// MaxID is the largest identifier possible to query.
var MaxID int

func init() {
	MaxID = getMaxID(config.SongDB)
}

// Aggressively fail on error
func checkErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// Query SQLite database to count maximum amount of rows, as to not point to non-existent ID
func getMaxID(database string) int {
	if _, err := os.Stat(config.SongDB); os.IsNotExist(err) {
		return 0
	}
	db, err := sql.Open("sqlite3", database)
	defer checkErrPanic(db.Close())
	checkErrPanic(err)
	var count int
	err = db.QueryRow("select max(ROWID) from music;").Scan(&count)
	checkErrPanic(err)
	return count
}

// GetRandomTrackIDs asks the database for n random IDs from the database
func GetRandomTrackIDs(amount int) (idList []int) {
	if getMaxID(config.SongDB) == 0 {
		return
	}

	db, err := sql.Open("sqlite3", config.SongDB)
	defer checkErrPanic(db.Close())
	checkErrPanic(err)
	var rows *sql.Rows
	rows, err = db.Query("SELECT ROWID from music ORDER BY random() LIMIT ?", amount)
	checkErrPanic(err)
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			log.Fatalln("GetRandomTrackIDs failed to scan rows.")
		}
		idList = append(idList, id)
	}
	return
}

// GetTrackById returns the "Human" friendly output and raw path of a track by its ID
func GetTrackById(trackID int) (human, path string) {
	MaxID = getMaxID(config.SongDB)
	if trackID > MaxID || trackID < 1 {
		return "", ""
	}
	db, err := sql.Open("sqlite3", config.SongDB)
	checkErrPanic(err)
	defer checkErrPanic(db.Close())
	var artist, title, album string
	err = db.QueryRow("select path,artist,title,album from MUSIC where ROWID = ?", trackID).Scan(&path, &artist, &title, &album)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", ""
		}
	}
	checkErrPanic(err)

	human = artist + " - " + title
	return
}

func FindArtistTitle(Query string) []string {
	Query = fmt.Sprintf("%%%s%%", Query)
	rows := makeDbQuery(config.SongDB, "SELECT ROWID, * FROM music where (artist || \" \" || title)  LIKE ? LIMIT 25", Query)
	defer checkErrPanic(rows.Close())

	var rowID int
	var artist, album, title, path string
	var output []string
	for rows.Next() {
		err := rows.Scan(&rowID, &artist, &album, &title, &path)
		checkErrPanic(err)
		output = append(output, fmt.Sprintf("#%d | %s - %s (%s)", rowID, artist, title, album))
	}

	return output
}

// Helper Functions
func makeDbQuery(database, query string, args ...interface{}) *sql.Rows {
	db, err := sql.Open("sqlite3", database)
	checkErrPanic(err)
	defer checkErrPanic(db.Close())
	rows, err := db.Query(query, args...)
	checkErrPanic(err)

	// Don't forget to close in function where called.
	return rows
}
