package storage

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"

	"log"

	"my-smtp-proxy/data"

	"encoding/json"
)

// Sqlite represents Sqlite backed storage backend
type SqliteDB struct {
	DB    *sql.DB
	Table string
}

// CreateSqlite creates a SqliteDB backed storage backend
func CreateSqlite(file, table string) *SqliteDB {
	log.Printf("Connecting to Sqlite: %s\n", file)
	db, err := sql.Open("sqlite", file)
	if err != nil {
		log.Printf("Error connecting to Sqlite: %s", err)
		return nil
	}
	// defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id integer primary key asc, message JSON)", table))
	if err != nil {
		log.Printf("Failed creating table: %s", err)
		return nil
	}

	log.Printf("Created table %s", table)

	return &SqliteDB{
		DB:    db,
		Table: table,
	}
}

// Store stores a message in MongoDB and returns its storage ID
func (sqlite *SqliteDB) Store(m *data.Message) (string, error) {
	encoded, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error converting message to json: %s", err)
		return "", err
	}

	result, err := sqlite.DB.Exec(fmt.Sprintf("INSERT INTO %s (message) VALUES (?)", sqlite.Table), encoded)
	if err != nil {
		log.Printf("Error inserting message: %s", err)
		return "", err
	}

	id, _ := result.LastInsertId()

	return string(id), nil
}

// Count returns the number of stored messages
func (sqlite *SqliteDB) Count() int {
	var count int

	err := sqlite.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", sqlite.Table)).Scan(&count)
	if err != nil {
		log.Printf("Error counting message: %s", err)
		return -1
	}

	return count
}

// Search finds messages matching the query
func (sqlite *SqliteDB) Search(kind, query string, start, limit int) (*data.Messages, int, error) {
	messages := &data.Messages{}
	var count = 0

	var field = "message->>'Raw'->>'Data'"
	switch kind {
	case "to":
		field = "message->>'Raw'->>'To'"
	case "from":
		field = "message->>'Raw'->>'From'"
	}

	query = fmt.Sprintf("%%%s%%", query)
	log.Printf("Searching for %s: %s (%d, %d)", field, query, start, limit)

	rows, err := sqlite.DB.Query(
		fmt.Sprintf("SELECT message FROM %s WHERE %s LIKE ? ORDER BY id LIMIT ? OFFSET ?", sqlite.Table, field), query, limit, start)
	if err != nil {
		log.Printf("Error loading messages: %s", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var message string

		if err := rows.Scan(&message); err != nil {
			log.Printf("Issue with row ... %s", err)
		}

		// Parse the content of message into a data.Message
		msg := &data.Message{}
		err := json.Unmarshal([]byte(message), msg)
		if err != nil {
			log.Printf("Error parsing message: %s", err)
			continue
		}

		// Add msg to messages
		*messages = append(*messages, *msg)
	}

	err = sqlite.DB.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s WHERE %s LIKE ?", sqlite.Table, field), query).Scan(&count)
	if err != nil {
		log.Printf("Error counting message: %s", err)
		return messages, -1, err
	}

	return messages, count, nil
}

// List returns a list of messages by index
func (sqlite *SqliteDB) List(start int, limit int) (*data.Messages, error) {
	messages := &data.Messages{}

	rows, err := sqlite.DB.Query(fmt.Sprintf("SELECT message FROM %s ORDER BY id LIMIT ? OFFSET ?", sqlite.Table), limit, start)
	if err != nil {
		log.Printf("Error loading messages: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message string

		if err := rows.Scan(&message); err != nil {
			log.Printf("Issue with row ... %s", err)
		}

		// Parse the content of message into a data.Message
		msg := &data.Message{}
		err := json.Unmarshal([]byte(message), msg)
		if err != nil {
			log.Printf("Error parsing message: %s", err)
			continue
		}

		// Add msg to messages
		*messages = append(*messages, *msg)
	}

	return messages, nil
}

// DeleteOne deletes an individual message by storage ID
func (sqlite *SqliteDB) DeleteOne(id string) error {
	log.Printf("Deleting message %s", id)
	_, err := sqlite.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE message->>'ID' = ?", sqlite.Table), id)
	return err
}

// DeleteAll deletes all messages stored in MongoDB
func (sqlite *SqliteDB) DeleteAll() error {
	_, err := sqlite.DB.Exec(fmt.Sprintf("DELETE FROM %s", sqlite.Table))
	return err
}

// Load loads an individual message by storage ID
func (sqlite *SqliteDB) Load(id string) (*data.Message, error) {
	var message string

	err := sqlite.DB.QueryRow(fmt.Sprintf("SELECT message FROM %s WHERE message->>'ID' = ?", sqlite.Table), id).Scan(&message)

	if err != nil {
		log.Printf("Error loading message: %s", err)
		return nil, err
	}

	msg := &data.Message{}
	err = json.Unmarshal([]byte(message), msg)
	if err != nil {
		log.Printf("Error loading message: %s", err)
		return nil, err
	}
	return msg, nil
}
