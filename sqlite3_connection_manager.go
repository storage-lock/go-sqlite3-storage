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
	// 注意：sql.Open 仅做驱动名校验与句柄分配（懒连接），"sqlite3" 驱动已由 mattn/go-sqlite3
	// 注册，此处恒返回 nil error。保留 error 返回值仅为兼容既有调用方签名。
	db, _ := sql.Open("sqlite3", dbPath)
	return &Sqlite3ConnectionManager{
		FixedSqlDBConnectionManager: storage.NewFixedSqlDBConnectionManager(db),
	}, nil
}

const SqliteConnectionManagerName = "sqlite-connection-manager"

func (x *Sqlite3ConnectionManager) Name() string {
	return SqliteConnectionManagerName
}
