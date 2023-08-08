package sqlite3_storage

import (
	"database/sql"
	"github.com/storage-lock/go-storage"
)

type Sqlite3StorageOptions struct {
	ConnectionManager storage.ConnectionManager[*sql.DB]
	TableName         string
}

func NewSqlite3StorageOptions() *Sqlite3StorageOptions {
	return &Sqlite3StorageOptions{
		TableName: storage.DefaultStorageTableName,
	}
}

func (x *Sqlite3StorageOptions) SetConnectionManager(connectionManager storage.ConnectionManager[*sql.DB]) *Sqlite3StorageOptions {
	x.ConnectionManager = connectionManager
	return x
}

func (x *Sqlite3StorageOptions) SetTableName(tableName string) *Sqlite3StorageOptions {
	x.TableName = tableName
	return x
}
