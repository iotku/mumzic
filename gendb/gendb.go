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
)

var dbfile = "./media.db"
var processednum = 0
var errorednum = 0
var removednum = 0

func main() {
	flag.Parse()
	path := flag.Arg(0)
	var sqltx *sql.Tx
	var database *sql.DB // Closed by scan/compare functions, I think. (unclear, but seems functional)

	if _, err := os.Stat(dbfile); os.IsNotExist(err) {
		// File doesn't exist, so do full DB run without comparision
		fmt.Println("Generate DB")
		sqltx, database = InitDB(dbfile)
		fullScan(path, sqltx)
	} else {
		database = openDB(dbfile)
		var count int
		err = database.QueryRow("SELECT COUNT(*) FROM music").Scan(&count)
		checkErr(err)
		fmt.Printf("DB Has %d Rows\n", count)
		sqltx, err = database.Begin()
		checkErr(err)
		if count == 0 {
			// Run full scan without checking database
			fullScan(path, sqltx)
		} else {
			// Run Comparison against recently added files
			compareDatabase(path, database, sqltx)
		}
	}

	err := sqltx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func openDB(dbfile string) *sql.DB {
	database, err := sql.Open("sqlite3", dbfile)
	checkErr(err)
	return database
}

func PrepareStatementInsert(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`INSERT INTO "music" (artist, album, title, path) VALUES (?, ?, ?, ?);`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func PrepareStatementRemove(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`DELETE FROM music WHERE path = ?`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func fullScan(path string, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx)
	defer stmt.Close()

	currentFiles := scanFiles(path)
	for _, v := range currentFiles {
		tags, err := getTags(v)
		if tags == nil {
			errorednum++
			printStatus("Error", err.Error()+" "+path)
			continue
		}
		addPathToDB(tags, stmt)
	}
}

// Recursively scan path for files to be added or compared to database
func scanFiles(path string) []string {
	var fileList []string
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".flac" {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		fmt.Println(err.Error())
	}
	return fileList
}

func getTags(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Println("Tag couldn't open path:", path)
		return nil, err
	}
	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	metadata := map[string]string{
		"artist": m.Artist(),
		"album":  m.Album(),
		"title":  m.Title(),
		"path":   path,
	}

	return metadata, nil
}

func compareDatabase(path string, database *sql.DB, tx *sql.Tx) {
	stmt := PrepareStatementInsert(tx)
	defer stmt.Close()

	// Is there a good way to do the comparison during the scan?
	currentFiles := scanFiles(path)

	var previousFiles []string
	previousFiles = loadOldFilesList(database)

	// Add these files
	filesToAdd := difference(currentFiles, previousFiles)

	for _, file := range filesToAdd {
		tags, err := getTags(file)
		if tags == nil {
			errorednum++
			printStatus("Error", err.Error()+" "+file)
			continue
		}
		addPathToDB(tags, stmt)
	}

	// remove these
	stmt = PrepareStatementRemove(tx)
	filesToRemove := difference(previousFiles, currentFiles)

	for _, file := range filesToRemove {
		fmt.Println(file)
		removePathFromDB(file, stmt)
	}

	fmt.Printf("\n%d:%d\n", len(currentFiles), len(previousFiles))
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func InitDB(dbfile string) (*sql.Tx, *sql.DB) {
	var db *sql.DB
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
	       PRAGMA secure_delete = OFF;
	       PRAGMA synchronous = OFF;
	       PRAGMA temp_store = MEMORY;
	       PRAGMA journal_mode = OFF;
	       PRAGMA wal_autocheckpoint = 16384;
	       CREATE TABLE IF NOT EXISTS "music" (
	           "artist" TEXT NOT NULL,
	           "album" TEXT NOT NULL,
	           "title" TEXT NOT NULL,
	           "path" TEXT NOT NULL
	       );
	       CREATE UNIQUE INDEX IF NOT EXISTS "path" ON "music" ("path");
	   `

	_, err = db.Exec(ddl)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	return tx, db
}

// Return the difference between two []string slices, TODO: is there a faster method?
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

func loadOldFilesList(database *sql.DB) []string {
	var files []string

	rows, err := database.Query("SELECT path FROM music")
	//defer rows.Close()
	checkErr(err)

	var path string
	for rows.Next() {
		err := rows.Scan(&path)
		files = append(files, path)
		checkErr(err)
	}
	return files
}

func addPathToDB(metadata map[string]string, stmt *sql.Stmt) {
	_, err := stmt.Exec(metadata["artist"], metadata["album"], metadata["title"], metadata["path"])
	if err != nil {
		// Early return if INSERT fails (hopefully because path already exists)
		log.Println(err)
		return
	}
	processednum++
	printStatus("Added", metadata["path"])
}

func removePathFromDB(path string, stmt *sql.Stmt) {
	_, err := stmt.Exec(path)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	removednum++
	printStatus("Removed", path)
}

func printStatus(action, path string) {
	fmt.Printf("Added: %d Error: %d Removed: %d | %s: %s\n", processednum, errorednum, removednum, action, path)
}
