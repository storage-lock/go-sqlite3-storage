package sqlite3_storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-infrastructure/go-iterator"
	_ "github.com/mattn/go-sqlite3"
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

	insertSql := fmt.Sprintf(`UPDATE %s SET version = ?, lock_information_json_string = ? WHERE lock_id = ? AND owner_id = ? AND version = ?`, x.options.TableName)
	execContext, err := db.ExecContext(ctx, insertSql, newVersion, lockInformation.ToJsonString(), lockId, lockInformation.OwnerId, exceptedVersion)
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
	rs, err := db.Query(getLockSql, lockId)
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
	rs, err := db.Query("SELECT STRFTIME('%s','now')")
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

	rows, err := db.Query("SELECT * FROM %s", x.options.TableName)
	if err != nil {
		return nil, err
	}
	return storage.NewSqlRowsIterator(rows), nil
}
