package search

import (
	"database/sql"
	"fmt"

	"layeh.com/gumble/gumble"
)

// Database generated from gendb
var songdb = "./media.db"

// Number of rows (not to exceed) in sqlite database
var MaxDBID = getMaxID(songdb)

// Aggresively fail on error
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Query SQLite database to count maximum amount of rows, as to not point to non existent ID
// TODO: Perhaps catch error instead?
func getMaxID(database string) int {
	db, err := sql.Open("sqlite3", database)
	defer db.Close()
	checkErr(err)
	var count int
	err = db.QueryRow("SELECT id FROM music WHERE ID = (SELECT MAX(ID) FROM music);").Scan(&count)
	checkErr(err)
	return count
}

// Query SQLite database to get filepath related to ID
func GetTrackById(trackID int) (filepath, humanout string) {
	if trackID > MaxDBID {
		return "", ""
	}
	db, err := sql.Open("sqlite3", songdb)
	checkErr(err)
	defer db.Close()
	var path, artist, title, album string
	err = db.QueryRow("select path,artist,title,album from MUSIC where id = ?", trackID).Scan(&path, &artist, &title, &album)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", ""
		}
	}
	checkErr(err)

	humanout = artist + " - " + title
	return path, humanout
}

func SearchALL(Query string, client *gumble.Client) {
	Query = fmt.Sprintf("%%%s%%", Query)
	rows := makeDbQuery(songdb, "SELECT * FROM music where (artist || \" \" || title)  LIKE ? LIMIT 25", Query)
	defer rows.Close()

	var id int
	var artist, album, title, path string

	for rows.Next() {
		err := rows.Scan(&id, &artist, &album, &title, &path)
		checkErr(err)
		chanMsg(client, fmt.Sprintf("#%d | %s - %s (%s)\n", id, artist, title, album))
	}

	return
}

// Helper Functions
func makeDbQuery(songdb, query string, args ...interface{}) *sql.Rows {
	db, err := sql.Open("sqlite3", songdb)
	checkErr(err)
	defer db.Close()
	rows, err := db.Query(query, args...)
	checkErr(err)

	// Don't forget to close in function where called.
	return rows
}

// TODO: consider handling sending messages outside of the search package for more proper segmentation
func chanMsg(client *gumble.Client, msg string) { client.Self.Channel.Send(msg, false) }
