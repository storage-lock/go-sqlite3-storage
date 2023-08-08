# SQLite3 Storage 

# 一、这是什么
以sqlite3为存储引擎的[Storage](https://github.com/storage-lock/go-storage)实现，当前仓库为比较底层的存储层实现，你可以与[storage-lock](https://github.com/storage-lock/go-storage-lock)结合使用。

sqlite3的数据库文件通常以.db或者.sqlite3结尾，可以使用[SQLiteSpy](https://download.cnet.com/SQLiteSpy/3000-2065_4-75451503.html)查看。

# 二、安装依赖
```bash
go get -u github.com/storage-lock/go-sqlite3-storage
```

# 三、API Examples

## 3.1 从数据库路径创建Sqlite3Storage

sqlite3是单文件数据库，连接的时候指定数据库文件的路径，下面的例子演示了如何从一个数据库文件路径创建Sqlite3Storage：

```go
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
```

## 3.2 从sql.DB创建Sqlite3Storage

或者现在你已经有从其它渠道创建的能够连接到sqlite3的sql.DB，则也可以从这个*sql.DB创建Sqlite3Storage：

```go
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

```





