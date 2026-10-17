package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

type SessionRecord struct {
	SessionID string    `json:"session_id"`
	JID       string    `json:"jid"`
	Phone     string    `json:"phone"`
	PushName  string    `json:"push_name"`
	Status    string    `json:"status"` // IDLE, SCAN_QR, CONNECTED, DISCONNECTED
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Database struct {
	DB        *sql.DB
	Container *sqlstore.Container
	Dialect   string
}

func Init(dialect, address string, log waLog.Logger) (*Database, error) {
	driverName := dialect
	sqlstoreDialect := dialect

	if strings.HasPrefix(dialect, "sqlite") {
		driverName = "sqlite"
		sqlstoreDialect = "sqlite3"
	} else if dialect == "postgres" || dialect == "postgresql" {
		driverName = "postgres"
		sqlstoreDialect = "postgres"
	}

	db, err := sql.Open(driverName, address)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ctx := context.Background()
	container := sqlstore.NewWithDB(db, sqlstoreDialect, log)
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("failed to upgrade whatsmeow database: %w", err)
	}

	d := &Database{
		DB:        db,
		Container: container,
		Dialect:   sqlstoreDialect,
	}

	if err := d.createSessionTable(); err != nil {
		return nil, fmt.Errorf("failed to create wa_sessions table: %w", err)
	}

	return d, nil
}

func (d *Database) createSessionTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS wa_sessions (
		session_id VARCHAR(100) PRIMARY KEY,
		jid VARCHAR(100) NOT NULL DEFAULT '',
		phone VARCHAR(50) NOT NULL DEFAULT '',
		push_name VARCHAR(100) NOT NULL DEFAULT '',
		status VARCHAR(50) NOT NULL DEFAULT 'IDLE',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := d.DB.Exec(query)
	return err
}

func (d *Database) UpsertSession(s *SessionRecord) error {
	now := time.Now()
	var query string

	if d.Dialect == "postgres" {
		query = `
		INSERT INTO wa_sessions (session_id, jid, phone, push_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (session_id) DO UPDATE SET
			jid = EXCLUDED.jid,
			phone = EXCLUDED.phone,
			push_name = EXCLUDED.push_name,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at;
		`
		_, err := d.DB.Exec(query, s.SessionID, s.JID, s.Phone, s.PushName, s.Status, now, now)
		return err
	}

	// SQLite syntax
	query = `
	INSERT INTO wa_sessions (session_id, jid, phone, push_name, status, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT (session_id) DO UPDATE SET
		jid = excluded.jid,
		phone = excluded.phone,
		push_name = excluded.push_name,
		status = excluded.status,
		updated_at = excluded.updated_at;
	`
	_, err := d.DB.Exec(query, s.SessionID, s.JID, s.Phone, s.PushName, s.Status, now, now)
	return err
}

func (d *Database) UpdateStatus(sessionID, status string) error {
	now := time.Now()
	var query string
	if d.Dialect == "postgres" {
		query = `UPDATE wa_sessions SET status = $1, updated_at = $2 WHERE session_id = $3`
		_, err := d.DB.Exec(query, status, now, sessionID)
		return err
	}

	query = `UPDATE wa_sessions SET status = ?, updated_at = ? WHERE session_id = ?`
	_, err := d.DB.Exec(query, status, now, sessionID)
	return err
}

func (d *Database) GetSession(sessionID string) (*SessionRecord, error) {
	var query string
	if d.Dialect == "postgres" {
		query = `SELECT session_id, jid, phone, push_name, status, created_at, updated_at FROM wa_sessions WHERE session_id = $1`
	} else {
		query = `SELECT session_id, jid, phone, push_name, status, created_at, updated_at FROM wa_sessions WHERE session_id = ?`
	}

	row := d.DB.QueryRow(query, sessionID)
	var s SessionRecord
	err := row.Scan(&s.SessionID, &s.JID, &s.Phone, &s.PushName, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) GetAllSessions() ([]*SessionRecord, error) {
	query := `SELECT session_id, jid, phone, push_name, status, created_at, updated_at FROM wa_sessions`
	rows, err := d.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SessionRecord
	for rows.Next() {
		var s SessionRecord
		if err := rows.Scan(&s.SessionID, &s.JID, &s.Phone, &s.PushName, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, nil
}

func (d *Database) DeleteSession(sessionID string) error {
	var query string
	if d.Dialect == "postgres" {
		query = `DELETE FROM wa_sessions WHERE session_id = $1`
	} else {
		query = `DELETE FROM wa_sessions WHERE session_id = ?`
	}
	_, err := d.DB.Exec(query, sessionID)
	return err
}
