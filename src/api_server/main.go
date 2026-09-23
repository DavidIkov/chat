package main

import (
	"chat/src/api_server/user"
	"flag"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"database/sql"
)

func main() {
	listenURL := flag.String("listenURL", "", "listen url")
	dbURL := flag.String("dbURL", "", "database url")
	flag.Parse()

	if *listenURL == "" {
		log.Fatal("addr cannot be empty")
	} else if *dbURL == "" {
		log.Fatal("dbURL cannot be empty")
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		log.Fatal(err)
	}

	usersManager, err := user.CreateUsersManager(db)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/user/register", usersManager.UserRegistrationHandler)
	mux.HandleFunc("/user/login", usersManager.UserLogInHandler)

	log.Println("Started listening on ", *listenURL)
	if err := http.ListenAndServe(*listenURL, mux); err != nil {
		log.Fatal(err)
	}
}
