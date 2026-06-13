package queue

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
)

// TaskQueue is a persistent SQLite store for outbound messages.
type TaskQueue struct {
	db *sql.DB
}

// QueuedMessage represents a message stored in the queue.
type QueuedMessage struct {
	ID        int
	TargetID  string
	Message   protocol.Message
	CreatedAt time.Time
}

// NewTaskQueue initializes a persistent SQLite-backed queue.
func NewTaskQueue(dbPath string) (*TaskQueue, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if it doesn't exist
	schema := `
	CREATE TABLE IF NOT EXISTS outbound_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_id TEXT NOT NULL,
		message_json TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &TaskQueue{db: db}, nil
}

// Enqueue stores a message persistently for later delivery.
func (q *TaskQueue) Enqueue(targetID string, msg protocol.Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = q.db.Exec("INSERT INTO outbound_queue (target_id, message_json) VALUES (?, ?)", targetID, string(msgBytes))
	if err != nil {
		log.Printf("⚠️ Failed to enqueue message for %s: %v", targetID, err)
		return err
	}
	return nil
}

// FetchAll retrieves all queued messages.
func (q *TaskQueue) FetchAll() ([]QueuedMessage, error) {
	rows, err := q.db.Query("SELECT id, target_id, message_json, created_at FROM outbound_queue ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []QueuedMessage
	for rows.Next() {
		var qm QueuedMessage
		var msgJSON string
		if err := rows.Scan(&qm.ID, &qm.TargetID, &msgJSON, &qm.CreatedAt); err != nil {
			continue
		}
		if err := json.Unmarshal([]byte(msgJSON), &qm.Message); err == nil {
			msgs = append(msgs, qm)
		}
	}
	return msgs, nil
}

// Delete removes a message from the queue after successful delivery.
func (q *TaskQueue) Delete(id int) error {
	_, err := q.db.Exec("DELETE FROM outbound_queue WHERE id = ?", id)
	return err
}

// Close closes the underlying database connection.
func (q *TaskQueue) Close() error {
	return q.db.Close()
}
