package main

import (
	"context"
	"database/sql"
	"fmt"
	sqlite3_storage "github.com/storage-lock/go-sqlite3-storage"
	storage "github.com/storage-lock/go-storage"
)

func main() {

	// 假设已经在其它地方初始化数据库连接得到了一个*sql.DB
	dbPath := "test_data/test.db3"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}

	// 则可以从这个*sql.DB中创建一个Sqlite3Storage
	connectionManager := storage.NewFixedSqlDBConnectionManager(db)
	options := sqlite3_storage.NewSqlite3StorageOptions().SetConnectionManager(connectionManager)
	storage, err := sqlite3_storage.NewSqlite3Storage(context.Background(), options)
	if err != nil {
		panic(err)
	}
	fmt.Println(storage.GetName())

}
