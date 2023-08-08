package sqlite3_storage

import (
	"database/sql"
	"github.com/storage-lock/go-storage"
)

// Sqlite3ConnectionManager 用于连接到sqlite3的数据库
type Sqlite3ConnectionManager struct {
	*storage.FixedSqlDBConnectionManager
}

var _ storage.ConnectionManager[*sql.DB] = &Sqlite3ConnectionManager{}

func NewSqlite3ConnectionManager(dbPath string) (*Sqlite3ConnectionManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return &Sqlite3ConnectionManager{
		FixedSqlDBConnectionManager: storage.NewFixedSqlDBConnectionManager(db),
	}, nil
}

const SqliteConnectionManagerName = "sqlite-connection-manager"

func (x *Sqlite3ConnectionManager) Name() string {
	return SqliteConnectionManagerName
}
