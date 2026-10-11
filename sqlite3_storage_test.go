package sqlite3_storage

import (
	"context"
	"errors"
	storage_test_helper "github.com/storage-lock/go-storage-test-helper"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const defaultTestDbPath = "storage_lock_test.db3"

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

// TestSqlite3Storage_CreateWithVersion_DuplicateMapsToLockAlreadyExists 验证回归：
// lock_id 是主键，并发/重复创建同一个锁时应命中主键唯一约束冲突。这时驱动返回的是
// *sqlite3.Error（Code == ErrConstraint），如果不显式映射，会被当成随机存储错误
// 直接抛给 core 层，导致 tryLock/tryRLock 无法识别"锁已存在，应重试/阻塞"这一
// 可重试条件，破坏分布式锁最基本的语义。
func TestSqlite3Storage_CreateWithVersion_DuplicateMapsToLockAlreadyExists(t *testing.T) {
	connectionManager, err := NewSqlite3ConnectionManager(defaultTestDbPath)
	assert.Nil(t, err)

	s, err := NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(connectionManager))
	assert.Nil(t, err)

	ctx := context.Background()
	lockId := "test-duplicate-create-lock-id"

	// 清理可能存在的历史记录
	_ = s.DeleteWithVersion(ctx, lockId, 0, &storage.LockInformation{})

	info := &storage.LockInformation{OwnerId: "owner-1", Version: 1}
	err = s.CreateWithVersion(ctx, lockId, 1, info)
	assert.Nil(t, err)
	defer func() { _ = s.DeleteWithVersion(ctx, lockId, 1, info) }()

	// 同一个 lock_id 再次创建，应命中主键唯一约束冲突，映射为 ErrLockAlreadyExists
	err = s.CreateWithVersion(ctx, lockId, 1, &storage.LockInformation{OwnerId: "owner-2", Version: 1})
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, storage_lock.ErrLockAlreadyExists), "期望 ErrLockAlreadyExists，实际: %v", err)
}
