package main

import (
	"chat/src/webui_server/handlers"
	"chat/src/webui_server/handlers/middleware"
	"chat/src/webui_server/handlers/render"
	"chat/src/webui_server/services"
	"flag"
	"log"
	"net/http"
	"os"
)

func main() {
	listenURL := flag.String("listenURL", "", "listen url")
	templatesDir := flag.String("templatesDir", "src/webui_server/templates", "directory with html templates")
	staticDir := flag.String("staticDir", "src/webui_server/static", "directory with static assets")
	flag.Parse()

	if *listenURL == "" {
		log.Fatal("listenURL cannot be empty")
	}

	services, err := services.CreateServices()
	if err != nil {
		log.Fatal(err)
	}

	templates, err := render.LoadTemplates(os.DirFS(*templatesDir))
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	if _, err = handlers.CreateHandlers(mux, services, templates, os.DirFS(*staticDir)); err != nil {
		log.Fatal(err)
	}

	log.Println("Started listening on ", *listenURL)
	if err := http.ListenAndServe(*listenURL, middleware.Recover(mux)); err != nil {
		log.Fatal(err)
	}
}
