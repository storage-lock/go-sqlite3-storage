package main

import (
	"context"
	"fmt"
	sqlite3_storage "github.com/storage-lock/go-sqlite3-storage"
)

func main() {

	// 使用一个sqlite3的数据库的路径创建ConnectionManager
	dbPath := "test_data/test.db3"
	connectionManager, err := sqlite3_storage.NewSqlite3ConnectionManager(dbPath)
	if err != nil {
		panic(err)
	}

	// 然后从这个ConnectionManager创建Sqlite3Storage
	options := sqlite3_storage.NewSqlite3StorageOptions().SetConnectionManager(connectionManager)
	storage, err := sqlite3_storage.NewSqlite3Storage(context.Background(), options)
	if err != nil {
		panic(err)
	}
	fmt.Println(storage.GetName())

}
