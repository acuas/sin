package db

import (
	"database/sql"
	"fmt"
	"log"
)

// Paste stores a pastes ID and its contents
type Paste struct {
	ID   string
	Data []byte
}

type PasteDatabase struct {
	db *sql.DB
}

func CreatePasteDatabase(name string) *PasteDatabase {
	pasteDB := &PasteDatabase{}

	db, err := sql.Open("mysql", fmt.Sprintf("root:example@tcp(127.0.0.1:3306)/%s", name))
	if err != nil {
		log.Fatalf("opening db: %s\n", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS Paste (
			id INTEGER NOT NULL,
			data BLOB NOT NULL, PRIMARY KEY(id))
	`)

	// TODO: Check err
	pasteDB.db = db

	return pasteDB
}

// RetrievePaste queries the database for the paste and returns it.
func (pasteDB *PasteDatabase) RetrievePaste(id string) (*Paste, error) {
	paste := &Paste{ID: id}
	if id == "favicon.ico" {
		return paste, nil
	}
	query := "SELECT data FROM Paste WHERE id=" + id
	rows, err := pasteDB.db.Query(query)
	response := []byte{}
	for rows.Next() {
		rows.Scan(&response)
		paste.Data = append(paste.Data, response...)
	}
	rows.Close()
	return paste, err
}

// StorePaste stores the paste in the database.
func (pasteDB *PasteDatabase) StorePaste(data []byte) (*Paste, error) {
	n := pasteDB.nextID()
	paste := &Paste{ID: intToID(n), Data: data}
	stmt, err := pasteDB.db.Prepare("INSERT INTO Paste (id, data) VALUES (?, ?)")
	if err != nil {
		return paste, fmt.Errorf("StorePaste preparing insert statement: %s", err)
	}
	_, err = stmt.Exec(n, data)
	if err != nil {
		return paste, fmt.Errorf("StorePaste executing insert: %s", err)
	}

	return paste, nil
}

func (pasteDB *PasteDatabase) nextID() uint64 {
	var i uint64
	rows, _ := pasteDB.db.Query("SELECT id FROM Paste ORDER BY id DESC")
	if rows.Next() {
		rows.Scan(&i)
		rows.Close()
		return i + 1
	}

	return 0
}
