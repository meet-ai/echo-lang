// Package config 测试 echo.lock 的读取与写入
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLock_NotExists(t *testing.T) {
	dir := t.TempDir()
	lf, err := LoadLock(dir)
	if err != nil {
		t.Fatalf("LoadLock() err = %v", err)
	}
	if lf != nil {
		t.Errorf("LoadLock() = %v, want nil when file not exists", lf)
	}
}

func TestSaveLock_LoadLock_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	lf := &LockFile{
		SchemaVersion: "1",
		Dependencies: map[string]string{
			"mathlib":       "2.1.0",
			"network/http": "1.0.0",
		},
	}
	if err := SaveLock(dir, lf); err != nil {
		t.Fatalf("SaveLock() err = %v", err)
	}
	path := filepath.Join(dir, "echo.lock")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("echo.lock was not created")
	}

	loaded, err := LoadLock(dir)
	if err != nil {
		t.Fatalf("LoadLock() err = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadLock() returned nil")
	}
	if loaded.SchemaVersion != "1" {
		t.Errorf("SchemaVersion = %q, want 1", loaded.SchemaVersion)
	}
	if v := loaded.GetLockedVersion("mathlib"); v != "2.1.0" {
		t.Errorf("GetLockedVersion(mathlib) = %q, want 2.1.0", v)
	}
	if v := loaded.GetLockedVersion("network/http"); v != "1.0.0" {
		t.Errorf("GetLockedVersion(network/http) = %q, want 1.0.0", v)
	}
}

func TestLockFile_GetSetLockedVersion(t *testing.T) {
	lf := &LockFile{Dependencies: make(map[string]string)}
	if v := lf.GetLockedVersion("pkg"); v != "" {
		t.Errorf("GetLockedVersion(empty) = %q", v)
	}
	lf.SetLockedVersion("pkg", "1.0.0")
	if v := lf.GetLockedVersion("pkg"); v != "1.0.0" {
		t.Errorf("GetLockedVersion(pkg) = %q, want 1.0.0", v)
	}
}
