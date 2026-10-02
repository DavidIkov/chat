package apitest

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"chat/internal/api_server/services"

	_ "github.com/lib/pq"
)

// defaultTestDBURL matches dev/docker-compose.yml. It points at the "postgres"
// maintenance database on purpose: TestMain creates (and later drops) its own
// throwaway database, so the dev "chatdb" is never touched.
const defaultTestDBURL = "postgres://admin:admin@localhost:7337/postgres?sslmode=disable"

var (
	adminDB    *sql.DB // maintenance connection used to create/drop the test database
	testDB     *sql.DB // connection to the throwaway test database
	testDBName string
)

func TestMain(m *testing.M) {
	flag.Parse()

	// -short skips every database-backed test, so there is nothing to set up.
	if testing.Short() {
		os.Exit(m.Run())
	}

	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = defaultTestDBURL
	}

	if err := startTestDatabase(dbURL); err != nil {
		fatalf("%v", err)
	}

	code := m.Run()

	dropTestDatabase()
	os.Exit(code)
}

// startTestDatabase connects to the maintenance database behind dbURL, creates a
// uniquely named database, connects to it and creates the schema through the
// production constructor so tests never drift from the real DDL.
func startTestDatabase(dbURL string) error {
	admin, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("open maintenance database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		return fmt.Errorf("cannot reach Postgres at TEST_DB_URL=%q: %w\n"+
			"\tstart it first, e.g. `make pgup` or `cd dev && docker compose up -d --wait`", dbURL, err)
	}
	adminDB = admin

	testDBName = fmt.Sprintf("chatdb_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, `create database "`+testDBName+`"`); err != nil {
		admin.Close()
		return fmt.Errorf("create test database %q: %w", testDBName, err)
	}

	dsn, err := replaceDatabaseName(dbURL, testDBName)
	if err != nil {
		return err
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open test database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping test database: %w", err)
	}
	if _, err := services.CreateServices(db); err != nil {
		db.Close()
		return fmt.Errorf("create schema: %w", err)
	}
	testDB = db
	return nil
}

func dropTestDatabase() {
	if testDB != nil {
		testDB.Close()
	}
	if adminDB == nil {
		return
	}
	if testDBName != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = adminDB.ExecContext(ctx, `drop database if exists "`+testDBName+`"`)
		cancel()
	}
	adminDB.Close()
}

// replaceDatabaseName swaps the database (path) component of a postgres URL.
func replaceDatabaseName(rawURL, name string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("TEST_DB_URL is not a valid URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("TEST_DB_URL must be a postgres:// URL, got %q", rawURL)
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "apitest: "+format+"\n", args...)
	os.Exit(1)
}
