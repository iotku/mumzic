package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"path/filepath"
	// "time"
)

var id = 0
var dbfile = "./media.db"
var lastid = 0
var processednum = 0
var errorednum = 0
var removednum = 0

var db *sql.DB
var txg *sql.Tx


func main() {
	flag.Parse()
	root := flag.Arg(0)
	currentFiles := []string{}
	var previousFiles []string
	previousFiles = loadOldFilesList()
	InitDB(dbfile)
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".flac" {
			currentFiles = append(currentFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}

	// Add these files
	filesToAdd := difference(currentFiles, previousFiles)

	for _, file := range filesToAdd {
		getTags(file)
	}

	// remove these
	filesToRemove := difference(previousFiles, currentFiles)

	for _, file := range filesToRemove {
		removePathFromDB(file)
	}

	fmt.Printf("\n %d:%d\n", len(currentFiles), len(previousFiles))
	err = txg.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func checkshrink() {
	id = id + 1
	if id > 30000 {
		txg.Commit()
		db, err := sql.Open("sqlite3", dbfile)
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec(`PRAGMA shrink_memory;`)
		if err != nil {
			log.Fatal(err)
		}
		// time.Sleep(5000)

		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}
		id = 0
		txg = tx
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func InitDB(dbfile string) {
	db, err := sql.Open("sqlite3", dbfile)
	checkErr(err)

	ddl := `
	       PRAGMA automatic_index = ON;
	       PRAGMA cache_size = 32768;
	       PRAGMA cache_spill = OFF;
	       PRAGMA foreign_keys = ON;
	       PRAGMA journal_size_limit = 67110000;
	       PRAGMA locking_mode = NORMAL;
	       PRAGMA page_size = 4096;
	       PRAGMA recursive_triggers = ON;
	       PRAGMA secure_delete = ON;
	       PRAGMA synchronous = OFF;
	       PRAGMA temp_store = MEMORY;
	       PRAGMA journal_mode = OFF;
	       PRAGMA wal_autocheckpoint = 16384;

	       CREATE TABLE IF NOT EXISTS "music" (
	           "id" INTEGER NOT NULL,
	           "artist" TEXT NOT NULL,
	           "album" TEXT NOT NULL,
	           "title" TEXT NOT NULL,
	           "path" TEXT NOT NULL
	       );

	       CREATE UNIQUE INDEX IF NOT EXISTS "id" ON "music" ("id");
	       CREATE UNIQUE INDEX IF NOT EXISTS "path" ON "music" ("path");
	   `

	_, err = db.Exec(ddl)
	if err != nil {
		log.Fatal(err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM music").Scan(&count)
	checkErr(err)
	fmt.Printf("DB Has %d Rows\n", count)
	lastid = count + 1

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	txg = tx

	if err != nil {
		log.Fatal(err)
	}
}

func getTags(thepath string) {
	f, err := os.Open(thepath)
	if err != nil {
		fmt.Println("error1: " + err.Error())
	}
	m, err := tag.ReadFrom(f)
	if err != nil {
		printStatus("Error", err.Error()+" "+thepath)
		errorednum++
		return
	}
	defer f.Close()

	track := map[string]string{
		"artist": m.Artist(),
		"album":  m.Album(),
		"title":  m.Title(),
		"path":   thepath,
	}

	stmt, err := txg.Prepare(`INSERT INTO "music" (id, artist, album, title, path) VALUES (?, ?, ?, ?, ?);`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	// fmt.Println(lastid)
	_, err = stmt.Exec(lastid, track["artist"], track["album"], track["title"], track["path"])
	if err != nil {
		// Early return if INSERT fails (hopefully because path already exists)
		// fmt.Println(err.Error())
		return
	}
	lastid++
	processednum++
	checkshrink()
	printStatus("Added", thepath)
}

func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

func loadOldFilesList() []string {
	var files []string
	if _, err := os.Stat(dbfile); os.IsNotExist(err) {
		// fmt.Printf("file does not exist")
		return files
	}

	olddb, err := sql.Open("sqlite3", dbfile)
	checkErr(err)
	defer olddb.Close()

	rows, err := olddb.Query("SELECT path FROM music")
	checkErr(err)
	defer rows.Close()
	var path string
	for rows.Next() {
		err := rows.Scan(&path)
		files = append(files, path)
		checkErr(err)
	}
	return files
}

func removePathFromDB(path string) {
	stmt, err := txg.Prepare(`DELETE FROM music WHERE path = ?`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	// fmt.Println(lastid)
	_, err = stmt.Exec(path)
	if err != nil {
		// Early return if INSERT fails (hopefully because path already exists)
		// fmt.Println(err.Error())
		fmt.Println(err.Error())
		return
	}
	removednum++
	printStatus("Removed", path)
	checkshrink()
}

func printStatus(action, path string) {
	fmt.Printf("Added: %d Error: %d Removed: %d | %s: %s\n", processednum, errorednum, removednum, action, path)
}
