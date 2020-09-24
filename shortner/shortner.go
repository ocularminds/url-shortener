package shortner

import (
	"log"
	"net/http"
        "strconv"
	"github.com/gorilla/mux"
)

var config Config;

func setStaticFolder(route *mux.Router) {
	fs := http.FileServer(http.Dir("./public/"))
	route.PathPrefix("/public/").Handler(http.StripPrefix("/public/", fs))
}

// BuildRoutes will add the routes for the application
func BuildRoutes(cfg Config) {
        config = cfg
	log.Println("Server will start at http://localhost:",strconv.Itoa(config.Port),"/")
	log.Println("Building Routes...")
	route := mux.NewRouter().StrictSlash(true)
	setStaticFolder(route)
	fs := http.FileServer(http.Dir("./views/"))
	route.PathPrefix("/files/").Handler(http.StripPrefix("/files/", fs))
	route.HandleFunc("/{slug}", HandleGetLink).Methods("GET")
	route.HandleFunc("/", HandlePostLink).Methods("POST")
	route.HandleFunc("/", HandleHomeLink).Methods("GET")
	log.Println("Routes are Loaded.")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), route))
}
