package mathapp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

var ErrNotFound = errors.New("没有找到这次体检，请回到首页重新开始")

func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (id TEXT PRIMARY KEY, owner TEXT NOT NULL, start_key TEXT NOT NULL, state TEXT NOT NULL, created INTEGER NOT NULL, UNIQUE(owner,start_key)); CREATE INDEX IF NOT EXISTS sessions_owner_created ON sessions(owner, created DESC);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Start(owner string, content Content, modules []string, key string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var id string
	err := s.db.QueryRow(`SELECT id FROM sessions WHERE owner=? AND start_key=?`, owner, key).Scan(&id)
	if err == nil {
		return s.Get(owner, id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Session{}, err
	}
	count, err := s.Count(owner)
	if err != nil {
		return Session{}, err
	}
	session, err := NewSession(content, modules, count, key)
	if err != nil {
		return Session{}, err
	}
	b, err := json.Marshal(session)
	if err != nil {
		return Session{}, err
	}
	_, err = s.db.Exec(`INSERT INTO sessions(id,owner,start_key,state,created) VALUES(?,?,?,?,?)`, session.ID, owner, key, string(b), session.Created)
	return session, err
}
func (s *Store) Get(owner, id string) (Session, error) {
	var value string
	err := s.db.QueryRow(`SELECT state FROM sessions WHERE owner=? AND id=?`, owner, id).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	var session Session
	err = json.Unmarshal([]byte(value), &session)
	return session, err
}
func (s *Store) Latest(owner string) (Session, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM sessions WHERE owner=? ORDER BY created DESC, rowid DESC LIMIT 1`, owner).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	return s.Get(owner, id)
}
func (s *Store) Count(owner string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE owner=?`, owner).Scan(&count)
	return count, err
}

// Read, mutate and write in one transaction; parallel retries cannot add evidence.
func (s *Store) Update(owner, id string, mutate func(*Session) error) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	var b string
	if err = tx.QueryRow(`SELECT state FROM sessions WHERE owner=? AND id=?`, owner, id).Scan(&b); errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	} else if err != nil {
		return Session{}, err
	}
	var session Session
	if err = json.Unmarshal([]byte(b), &session); err != nil {
		return Session{}, err
	}
	if err = mutate(&session); err != nil {
		return Session{}, err
	}
	encoded, err := json.Marshal(session)
	if err != nil {
		return Session{}, err
	}
	if _, err = tx.Exec(`UPDATE sessions SET state=? WHERE owner=? AND id=?`, string(encoded), owner, id); err != nil {
		return Session{}, err
	}
	return session, tx.Commit()
}
