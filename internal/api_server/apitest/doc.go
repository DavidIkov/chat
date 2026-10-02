// Package apitest holds black-box tests for the api_server.
//
// The HTTP tests run the real router (handlers.NewServer) on top of a real
// PostgreSQL database and drive it with actual HTTP requests, so one test
// exercises the handler, service and SQL layers together. A handful of direct
// service tests cover edge cases (message pagination cursors, join-link
// lifetime and use counting, leaving the last member, session invalidation) that
// are awkward or slow to reach over HTTP.
//
// TestMain owns the database: it connects to TEST_DB_URL (default: the dev
// docker-compose Postgres, reachable via `make pgup`), creates a uniquely named
// throwaway database, and drops it when the run ends. TEST_DB_URL must be a
// postgres:// URL whose role is allowed to CREATE DATABASE (the compose
// POSTGRES_USER is a superuser, so the provided default works). The dev "chatdb"
// database is never touched.
//
// Run the whole suite with `make test`; `go test -short ./...` (or `make
// test-short`) skips everything that needs a database.
package apitest
