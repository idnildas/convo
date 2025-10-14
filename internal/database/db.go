package database

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"time"
	"path/filepath"
	"fmt"
	"sort"	

)

var DB *sql.DB

func GetDB() *sql.DB {
    return DB
}

func Connect(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil { return err }
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil { return err }

	DB = db
	log.Println("✅ MySQL connected")
	return nil
}

func RunMigrations(migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
    if err != nil {
        return fmt.Errorf("failed to read migrations: %w", err)
    }

    // ensure files run in order: 001 -> 002 -> 003
    sort.Strings(files)

	for _, file := range files {
    b, err := ioutil.ReadFile(file)
    if err != nil {
        return fmt.Errorf("failed to read migration %s: %w", file, err)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if _, err := DB.ExecContext(ctx, string(b)); err != nil {
        return fmt.Errorf("migration %s failed: %w", file, err)
    }
    log.Printf("✅ migration applied: %s", file)
}

return nil
}
