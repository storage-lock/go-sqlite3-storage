package main

import (
	"os"
	"testing"
)

func TestMainRuns(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()
	// main 使用相对路径 test_data/test.db3，预建目录，在临时目录下落库不污染仓库
	if err := os.MkdirAll("test_data", 0755); err != nil {
		t.Fatalf("创建 test_data 目录失败: %v", err)
	}
	main()
}
