package main

import (
	"chat/internal/api_server/handlers"
	"chat/internal/api_server/handlers/middleware"
	"chat/internal/api_server/services"
	"database/sql"
	"flag"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

func main() {
	listenURL := flag.String("listenURL", "", "listen url")
	dbURL := flag.String("dbURL", "", "database url")
	flag.Parse()

	if *listenURL == "" {
		log.Fatal("listenURL cannot be empty")
	} else if *dbURL == "" {
		log.Fatal("dbURL cannot be empty")
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		log.Fatal(err)
	}

	services, err := services.CreateServices(db)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	_, err = handlers.CreateHandlers(mux, services)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Started listening on ", *listenURL)
	if err := http.ListenAndServe(*listenURL, middleware.Recover(mux)); err != nil {
		log.Fatal(err)
	}
}
