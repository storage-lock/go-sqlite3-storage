package sqlite3_storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-infrastructure/go-iterator"
	"github.com/mattn/go-sqlite3"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"time"
)

type Sqlite3Storage struct {
	options *Sqlite3StorageOptions
}

var _ storage.Storage = &Sqlite3Storage{}

func NewSqlite3Storage(ctx context.Context, options *Sqlite3StorageOptions) (*Sqlite3Storage, error) {

	if options.ConnectionManager == nil {
		return nil, fmt.Errorf("connection manager nil")
	}

	storage := &Sqlite3Storage{
		options: options,
	}

	if err := storage.Init(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

const SqliteStorageName = "sqlite3-storage"

func (x *Sqlite3Storage) GetName() string {
	return SqliteStorageName
}

// Capabilities 声明 SQLite3 存储支持的能力
// SQLite3 通过主键唯一约束和 WHERE 条件更新支持 CAS，
// 通过 STRFTIME('%s','now') 提供数据库服务器时间
// 注意：SQLite3 是单文件数据库，主要用于单进程场景，多进程并发需要启用 WAL 模式
func (x *Sqlite3Storage) Capabilities() []storage.StorageCapability {
	return []storage.StorageCapability{
		storage.CapabilityCAS,
		storage.CapabilityReliableTime,
		storage.CapabilityWatch,
	}
}

func (x *Sqlite3Storage) Init(ctx context.Context) (returnError error) {

	if x.options.TableName == "" {
		x.options.TableName = storage.DefaultStorageTableName
	}

	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	// 仅当在表不存在的情况下才创建，如果已经存在的话则不管了
	createTableSql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    lock_id VARCHAR(255) NOT NULL PRIMARY KEY,
    owner_id VARCHAR(255) NOT NULL,
    version BIGINT NOT NULL,
    lock_information_json_string VARCHAR(255) NOT NULL
)`, x.options.TableName)
	stmt, err := db.Prepare(createTableSql)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	return err
}

func (x *Sqlite3Storage) UpdateWithVersion(ctx context.Context, lockId string, exceptedVersion, newVersion storage.Version, lockInformation *storage.LockInformation) (returnError error) {

	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	// 注意：WHERE 条件只校验 lock_id + version，不校验 owner_id。
	// 核心层抢占过期/墓碑记录时（rlockReclaim/lockExpired），传入的是新持有者的
	// lockInformation，若加 owner_id 条件则永远匹配不上旧记录，抢占死循环。
	// 对齐 sql-based 基类 Sql92Provider.UpdateWithVersionSql 的行为。
	insertSql := fmt.Sprintf(`UPDATE %s SET version = ?, lock_information_json_string = ? WHERE lock_id = ? AND version = ?`, x.options.TableName)
	execContext, err := db.ExecContext(ctx, insertSql, newVersion, lockInformation.ToJsonString(), lockId, exceptedVersion)
	if err != nil {
		return err
	}
	affected, err := execContext.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return storage_lock.ErrVersionMiss
	}
	return nil
}

func (x *Sqlite3Storage) CreateWithVersion(ctx context.Context, lockId string, version storage.Version, lockInformation *storage.LockInformation) (returnError error) {
	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	insertSql := fmt.Sprintf(`INSERT INTO %s (lock_id, owner_id, version, lock_information_json_string) VALUES (?, ?, ?, ?)`, x.options.TableName)
	execContext, err := db.ExecContext(ctx, insertSql, lockId, lockInformation.OwnerId, version, lockInformation.ToJsonString())
	if err != nil {
		// lock_id 是主键，并发创建时后到者会触发主键唯一约束冲突，
		// 这是"锁已存在"而不是随机存储错误，必须映射为 ErrLockAlreadyExists，
		// 否则核心层 tryLock/tryRLock 不认这是可重试条件，直接把冲突当成硬错误返回，
		// 分布式锁最基本的"后来者等待/重试"语义在 sqlite3 上就会失效。
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return storage_lock.ErrLockAlreadyExists
		}
		return err
	}
	affected, err := execContext.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return storage_lock.ErrVersionMiss
	}
	return nil
}

func (x *Sqlite3Storage) DeleteWithVersion(ctx context.Context, lockId string, exceptedVersion storage.Version, lockInformation *storage.LockInformation) (returnError error) {
	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	deleteSql := fmt.Sprintf(`DELETE FROM %s WHERE lock_id = ? AND owner_id = ? AND version = ?`, x.options.TableName)
	execContext, err := db.ExecContext(ctx, deleteSql, lockId, lockInformation.OwnerId, exceptedVersion)
	if err != nil {
		return err
	}
	affected, err := execContext.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return storage_lock.ErrVersionMiss
	}
	return nil
}

func (x *Sqlite3Storage) Get(ctx context.Context, lockId string) (lockInformationJsonString string, returnError error) {
	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	getLockSql := fmt.Sprintf("SELECT lock_information_json_string FROM %s WHERE lock_id = ?", x.options.TableName)
	rs, err := db.QueryContext(ctx, getLockSql, lockId)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = rs.Close()
	}()
	if !rs.Next() {
		return "", storage_lock.ErrLockNotFound
	}
	err = rs.Scan(&lockInformationJsonString)
	if err != nil {
		return "", err
	}
	return lockInformationJsonString, nil
}

func (x *Sqlite3Storage) GetTime(ctx context.Context) (now time.Time, returnError error) {

	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	var zero time.Time
	rs, err := db.QueryContext(ctx, "SELECT STRFTIME('%s','now')")
	if err != nil {
		return zero, err
	}
	defer func() {
		err := rs.Close()
		if returnError == nil {
			returnError = err
		}
	}()
	if !rs.Next() {
		return zero, errors.New("rs server time failed")
	}
	var databaseTimestamp uint64
	err = rs.Scan(&databaseTimestamp)
	if err != nil {
		return zero, err
	}

	// TODO 时区
	return time.Unix(int64(databaseTimestamp), 0), nil
}

func (x *Sqlite3Storage) Close(ctx context.Context) error {
	// 没啥要干的事...
	return nil
}

func (x *Sqlite3Storage) List(ctx context.Context) (lockInformation iterator.Iterator[*storage.LockInformation], returnError error) {

	db, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := x.options.ConnectionManager.Return(ctx, db)
		if returnError == nil {
			returnError = err
		}
	}()

	listSql := fmt.Sprintf("SELECT lock_information_json_string FROM %s", x.options.TableName)
	rows, err := db.QueryContext(ctx, listSql)
	if err != nil {
		return nil, err
	}
	return storage.NewSqlRowsIterator(rows), nil
}

func (x *Sqlite3Storage) Watch(ctx context.Context, lockId string) (<-chan *storage.LockInformation, error) {
	return storage.NewPollingLockWatcher(x, 500*time.Millisecond).Watch(ctx, lockId)
}
