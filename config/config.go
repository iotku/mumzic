package config

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

// Max amount of lines you wish commands to output (before hopefully, going into an unimplemented more buffer)
var MaxLines = 10

// Database generated from gendb
var Songdb = "./media.db"

// Playback Volume level
var VolumeLevel float32 = 0.25
var CmdPrefix = "!"
var LastChannel string // last channel bot was in
var configPath = "./config.db"
var Hostname string // Set by LoadConfig for no good reason

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// SQLite fun
func LoadConfig(hostname string) {
	Hostname = hostname // hack to set Hostname
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Save new config.db
		tx, _ := initConfigDB(configPath)
		stmt := PrepareStatementInsert(tx)
		defer stmt.Close()
		writeConfigToDB(hostname, stmt)
		err := tx.Commit()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		db := openDB(configPath)
		defer db.Close()
		row := db.QueryRow("SELECT * FROM config where Hostname = ?", hostname)
		err := row.Scan(&hostname, &VolumeLevel, &LastChannel, &CmdPrefix, &Songdb)
		checkErr(err)
	}
}

// SaveConfig writes the current configuraiton to the configuration sqlite database
func SaveConfig() {
	fmt.Println("Writing configuration to disk")
	db := openDB(configPath)
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt := PrepareUpdate(tx)
	defer stmt.Close()
	// There must be a better way to do this, but I'm tired and this will do for now.
	_, err = stmt.Exec(VolumeLevel, LastChannel, CmdPrefix, Hostname)
	checkErr(err)
	err = tx.Commit()
	checkErr(err)
}

func PrepareUpdate(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`UPDATE config SET VolumeLevel = ?, LastChannel = ?, CmdPrefix = ? WHERE Hostname = ?;`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func PrepareStatementInsert(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`INSERT INTO "config" (Hostname, VolumeLevel, LastChannel, CmdPrefix, SongDB) VALUES (?, ?, ?, ?, ?);`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func writeConfigToDB(hostname string, stmt *sql.Stmt) {
	_, err := stmt.Exec(hostname, VolumeLevel, LastChannel, CmdPrefix, Songdb)
	if err != nil {
		log.Fatalln("Writing Config Failed!:", err.Error())
	}
}

func openDB(dbfile string) *sql.DB {
	database, err := sql.Open("sqlite3", dbfile)
	checkErr(err)
	return database
}

func initConfigDB(dbfile string) (*sql.Tx, *sql.DB) {
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
	       CREATE TABLE IF NOT EXISTS "config" (
	           "Hostname" TEXT NOT NULL,
	           "VolumeLevel" TEXT NOT NULL,
	           "LastChannel" TEXT NOT NULL,
	           "CmdPrefix" TEXT NOT NULL,
	           "SongDB" TEXT NOT NULL
	       );
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
