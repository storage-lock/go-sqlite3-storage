package sqlite3_storage

import (
	"context"
	storage_test_helper "github.com/storage-lock/go-storage-test-helper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const defaultTestDbPath = "test_data/storage_lock_test.db3"

func TestNewMySQLStorage(t *testing.T) {
	dbPath := os.Getenv("STORAGE_LOCK_SQLITE3_DB_PATH")
	if dbPath == "" {
		dbPath = defaultTestDbPath
	}
	connectionManager, err := NewSqlite3ConnectionManager(defaultTestDbPath)
	assert.Nil(t, err)

	s, err := NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(connectionManager))
	assert.Nil(t, err)
	storage_test_helper.TestStorage(t, s)
}
