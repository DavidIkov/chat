package main

import (
	"chat/src/api_server/user"
	"log"
	"net/http"
	"os"
)

func main() {
	usersManager, err := user.CreateUsersManager(nil)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/user/register", usersManager.UserRegistrationHandler)
}
