package search

import (
	"database/sql"
	"fmt"
	"github.com/iotku/mumzic/database"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

// Aggressively fail on error
func checkErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// GetRandomTrackIDs asks the database for n random IDs from the database
func GetRandomTrackIDs(amount int) (idList []int) {
	if database.GetMaxID() == 0 {
		return
	}

	var rows *sql.Rows
	rows, err := database.SongDB.Query("SELECT ROWID from music ORDER BY random() LIMIT ?", amount)
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
	if trackID > database.GetMaxID() || trackID < 1 {
		return "", ""
	}
	var artist, title, album string
	err := database.SongDB.QueryRow("SELECT path,artist,title,album from music where ROWID = ?", trackID).Scan(&path, &artist, &title, &album)
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
	rows := makeDbQuery("SELECT ROWID, * FROM music where (artist || \" \" || title)  LIKE ? LIMIT 25", Query)

	var rowID int
	var artist, album, title, path string
	var output []string
	for rows.Next() {
		err := rows.Scan(&rowID, &artist, &album, &title, &path)
		checkErrPanic(err)
		output = append(output, fmt.Sprintf("#%d | %s - %s (%s)", rowID, artist, title, album))
	}
	checkErrPanic(rows.Close())

	return output
}

// Helper Functions
func makeDbQuery(query string, args ...interface{}) *sql.Rows {
	rows, err := database.SongDB.Query(query, args...)
	checkErrPanic(err)

	// Don't forget to close in function where called.
	return rows
}
