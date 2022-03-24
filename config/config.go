package config

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Volume   float32 // Volume for bot
	Prefix   string  // Prefix for commands in channel chat
	Channel  string  // Current ChannelName the bot is occupying
	Hostname string  // Hostname of connected server
	Username string  // Username of bot
}

// Max amount of lines you wish commands to output (before hopefully, going into an unimplemented more buffer)
var MaxLines = 10

// Database generated from gendb
var Songdb = "./media.db"

// Path to configuration db
var configPath = "./config.db"
var database *sql.DB

func init() {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		tx, _ := initConfigDB(configPath)
		checkErr(tx.Commit())
	}

	database = openDB(configPath)
}

func CloseDatabase() {
	checkErr(database.Close())
}

func NewConfig(hostname string) *Config {
	defaultConfig := Config{
		Volume:   0.3,
		Prefix:   "!",
		Channel:  "",
		Hostname: hostname,
	}

	var config Config
	row := database.QueryRow("SELECT * FROM config where Hostname = ?", hostname)
	err := row.Scan(&config.Hostname, &config.Volume, &config.Channel, &config.Prefix, &Songdb)
	if err != nil && err == sql.ErrNoRows { // create new configuration
		tx, _ := database.Begin()
		writeConfigToDB(defaultConfig, prepareStatementInsert(tx))
		checkErr(tx.Commit())
		return &defaultConfig
	} else if err != nil {
		panic("NewConfig failed")
	}

	return &config
}

// SaveConfig writes the current configuraiton to the configuration sqlite database
func (config *Config) Save() { // TODO: verify this is actually working
	log.Println("Writing configuration to disk")
	tx, err := database.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt := prepareUpdate(tx)
	defer stmt.Close()
	// There must be a better way to do this, but I'm tired and this will do for now.
	_, err = stmt.Exec(config.Volume, config.Channel, config.Prefix, config.Hostname)
	checkErr(err)
	checkErr(tx.Commit())
}

func prepareUpdate(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`UPDATE config SET VolumeLevel = ?, LastChannel = ?, CmdPrefix = ? WHERE Hostname = ?;`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func prepareStatementInsert(tx *sql.Tx) *sql.Stmt {
	stmt, err := tx.Prepare(`INSERT INTO "config" (Hostname, VolumeLevel, LastChannel, CmdPrefix, SongDB) VALUES (?, ?, ?, ?, ?);`)
	if err != nil {
		log.Fatal(err)
	}
	return stmt
}

func writeConfigToDB(config Config, stmt *sql.Stmt) {
	_, err := stmt.Exec(config.Hostname, config.Volume, config.Channel, config.Prefix, Songdb)
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

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
