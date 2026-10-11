package sqlite3_storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	storage_test_helper "github.com/storage-lock/go-storage-test-helper"
	"github.com/stretchr/testify/assert"
)

func newSqliteCovInfo(t *testing.T) *storage.LockInformation {
	return storage_test_helper.BuildTestLockInformation(t)
}

// errTakeConnManager Take/Return 可注入故障的 ConnectionManager
type errTakeConnManager struct {
	takeErr   error
	returnErr error
	db        *sql.DB
}

func (x *errTakeConnManager) Name() string { return "err-conn-manager" }
func (x *errTakeConnManager) Take(ctx context.Context) (*sql.DB, error) {
	if x.takeErr != nil {
		return nil, x.takeErr
	}
	return x.db, nil
}
func (x *errTakeConnManager) Return(ctx context.Context, conn *sql.DB) error {
	return x.returnErr
}
func (x *errTakeConnManager) Shutdown(ctx context.Context) error { return nil }

func newSqlmockSqliteStorage(t *testing.T, configure func(sqlmock.Sqlmock)) *Sqlite3Storage {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	configure(mock)
	cm := storage.NewFixedSqlDBConnectionManager(db)
	return &Sqlite3Storage{options: &Sqlite3StorageOptions{ConnectionManager: cm, TableName: "locks"}}
}

func TestCoverageSqlite3ConnectionManagerName(t *testing.T) {
	cm, err := NewSqlite3ConnectionManager(":memory:")
	assert.Nil(t, err)
	assert.Equal(t, SqliteConnectionManagerName, cm.Name())
}

func TestCoverageSqlite3OptionsSetTableName(t *testing.T) {
	cm, err := NewSqlite3ConnectionManager(":memory:")
	assert.Nil(t, err)
	opts := NewSqlite3StorageOptions().SetConnectionManager(cm)
	assert.Equal(t, opts, opts.SetTableName("locks_cov"))
	assert.Equal(t, "locks_cov", opts.TableName)
}

func TestCoverageSqlite3CapabilitiesWatchCloseName(t *testing.T) {
	cm, err := NewSqlite3ConnectionManager(":memory:")
	assert.Nil(t, err)
	s, err := NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(cm))
	assert.Nil(t, err)
	assert.Contains(t, s.Capabilities(), storage.CapabilityCAS)
	assert.Contains(t, s.Capabilities(), storage.CapabilityReliableTime)
	ch, err := s.Watch(context.Background(), "lock-watch-cov")
	assert.Nil(t, err)
	assert.NotNil(t, ch)
	_, err = s.Watch(context.Background(), "")
	assert.NotNil(t, err)
	assert.Equal(t, SqliteStorageName, s.GetName())
	assert.Nil(t, s.Close(context.Background()))
}

func TestCoverageSqlite3NewErrors(t *testing.T) {
	// ConnectionManager 为 nil
	_, err := NewSqlite3Storage(context.Background(), &Sqlite3StorageOptions{})
	assert.NotNil(t, err)
	// Init 的 Take 失败
	_, err = NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(
		&errTakeConnManager{takeErr: errors.New("take boom")}))
	assert.NotNil(t, err)
	// Init 的 Prepare 失败
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	mock.ExpectPrepare("CREATE TABLE IF NOT EXISTS").WillReturnError(errors.New("prepare boom"))
	_, err = NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(
		storage.NewFixedSqlDBConnectionManager(db)))
	assert.NotNil(t, err)
	// Init 的 Exec 失败
	db2, mock2, err := sqlmock.New()
	assert.Nil(t, err)
	mock2.ExpectPrepare("CREATE TABLE IF NOT EXISTS")
	mock2.ExpectExec("CREATE TABLE IF NOT EXISTS").WillReturnError(errors.New("exec boom"))
	_, err = NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(
		storage.NewFixedSqlDBConnectionManager(db2)))
	assert.NotNil(t, err)
	// Init 的 Return 失败透传
	db3, mock3, err := sqlmock.New()
	assert.Nil(t, err)
	mock3.ExpectPrepare("CREATE TABLE IF NOT EXISTS")
	mock3.ExpectExec("CREATE TABLE IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = NewSqlite3Storage(context.Background(), NewSqlite3StorageOptions().SetConnectionManager(
		&errTakeConnManager{db: db3, returnErr: errors.New("return boom")}))
	assert.NotNil(t, err)
	// TableName 为空补默认值
	goodCM, err := NewSqlite3ConnectionManager(":memory:")
	assert.Nil(t, err)
	_, err = NewSqlite3Storage(context.Background(), &Sqlite3StorageOptions{ConnectionManager: goodCM})
	assert.Nil(t, err)
}

func TestCoverageSqlite3TakeErrors(t *testing.T) {
	ctx := context.Background()
	info := newSqliteCovInfo(t)
	wantErr := errors.New("take boom")
	s := &Sqlite3Storage{options: &Sqlite3StorageOptions{
		ConnectionManager: &errTakeConnManager{takeErr: wantErr}, TableName: "locks"}}
	assert.Equal(t, wantErr, s.UpdateWithVersion(ctx, "l", 1, 2, info))
	assert.Equal(t, wantErr, s.CreateWithVersion(ctx, "l", 1, info))
	assert.Equal(t, wantErr, s.DeleteWithVersion(ctx, "l", 1, info))
	_, err := s.Get(ctx, "l")
	assert.Equal(t, wantErr, err)
	_, err = s.GetTime(ctx)
	assert.Equal(t, wantErr, err)
	_, err = s.List(ctx)
	assert.Equal(t, wantErr, err)
}

func TestCoverageSqlite3ExecQueryErrors(t *testing.T) {
	ctx := context.Background()
	info := newSqliteCovInfo(t)
	wantErr := errors.New("db boom")

	s2 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("UPDATE").WillReturnError(wantErr)
	})
	assert.Equal(t, wantErr, s2.UpdateWithVersion(ctx, "l", 1, 2, info))

	s3 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("INSERT").WillReturnError(wantErr)
	})
	assert.Equal(t, wantErr, s3.CreateWithVersion(ctx, "l", 1, info))

	s4 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("DELETE").WillReturnError(wantErr)
	})
	assert.Equal(t, wantErr, s4.DeleteWithVersion(ctx, "l", 1, info))

	s5 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectQuery("SELECT lock_information_json_string FROM").WillReturnError(wantErr)
	})
	_, err := s5.Get(ctx, "l")
	assert.Equal(t, wantErr, err)

	s6 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		rows := sqlmock.NewRows([]string{"lock_information_json_string"})
		m.ExpectQuery("SELECT lock_information_json_string FROM").WillReturnRows(rows)
	})
	_, err = s6.Get(ctx, "l")
	assert.Equal(t, storage_lock.ErrLockNotFound, err)

	s7 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectQuery("SELECT STRFTIME").WillReturnError(wantErr)
	})
	_, err = s7.GetTime(ctx)
	assert.Equal(t, wantErr, err)

	s8 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		rows := sqlmock.NewRows([]string{"t"})
		m.ExpectQuery("SELECT STRFTIME").WillReturnRows(rows)
	})
	_, err = s8.GetTime(ctx)
	assert.NotNil(t, err)

	s9 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectQuery("SELECT lock_information_json_string FROM").WillReturnError(wantErr)
	})
	_, err = s9.List(ctx)
	assert.Equal(t, wantErr, err)
}

func TestCoverageSqlite3RowsAffectedErrors(t *testing.T) {
	ctx := context.Background()
	info := newSqliteCovInfo(t)
	wantErr := errors.New("rows boom")

	s10 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewErrorResult(wantErr))
	})
	assert.Equal(t, wantErr, s10.UpdateWithVersion(ctx, "l", 1, 2, info))

	s11 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("INSERT").WillReturnResult(sqlmock.NewErrorResult(wantErr))
	})
	assert.Equal(t, wantErr, s11.CreateWithVersion(ctx, "l", 1, info))

	s12 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("DELETE").WillReturnResult(sqlmock.NewErrorResult(wantErr))
	})
	assert.Equal(t, wantErr, s12.DeleteWithVersion(ctx, "l", 1, info))
}

func TestCoverageSqlite3ZeroRows(t *testing.T) {
	ctx := context.Background()
	info := newSqliteCovInfo(t)

	s13 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 0))
	})
	assert.Equal(t, storage_lock.ErrVersionMiss, s13.UpdateWithVersion(ctx, "l", 1, 2, info))

	s14 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(0, 0))
	})
	assert.Equal(t, storage_lock.ErrVersionMiss, s14.CreateWithVersion(ctx, "l", 1, info))

	s15 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		m.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 0))
	})
	assert.Equal(t, storage_lock.ErrVersionMiss, s15.DeleteWithVersion(ctx, "l", 1, info))
}

func TestCoverageSqlite3ScanErrors(t *testing.T) {
	ctx := context.Background()
	info := newSqliteCovInfo(t)
	_ = info

	s17 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		// NULL 无法 Scan 进 string，触发 rs.Scan 错误分支
		rows := sqlmock.NewRows([]string{"lock_information_json_string"}).AddRow(nil)
		m.ExpectQuery("SELECT lock_information_json_string FROM").WillReturnRows(rows)
	})
	_, err := s17.Get(ctx, "l")
	assert.NotNil(t, err)

	s18 := newSqlmockSqliteStorage(t, func(m sqlmock.Sqlmock) {
		// NULL 无法 Scan 进 uint64，触发 rs.Scan 错误分支
		rows := sqlmock.NewRows([]string{"t"}).AddRow(nil)
		m.ExpectQuery("SELECT STRFTIME").WillReturnRows(rows)
	})
	_, err = s18.GetTime(ctx)
	assert.NotNil(t, err)
}

func TestCoverageSqlite3ReturnErrorOnSuccess(t *testing.T) {
	ctx := context.Background()
	info := newSqliteCovInfo(t)
	wantErr := errors.New("return boom")
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(1, 1))
	s := &Sqlite3Storage{options: &Sqlite3StorageOptions{
		ConnectionManager: &errTakeConnManager{db: db, returnErr: wantErr}, TableName: "locks"}}
	assert.Equal(t, wantErr, s.UpdateWithVersion(ctx, "l", 1, 2, info))
}
