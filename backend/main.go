package main

import (
	"log"
	"net/http"
)

var listenAndServe = http.ListenAndServe

func main() {
	if err := run("./data/performance.db", ":8080"); err != nil {
		log.Fatal(err)
	}
}

func run(databasePath, address string) error {
	database, err := openDatabase(databasePath)
	if err != nil {
		return err
	}
	defer database.Close()

	db = database
	if resolved, rerr := resolveDatabasePath(databasePath); rerr == nil {
		log.Printf("Using database at %s", resolved)
	}
	log.Printf("API running on http://localhost%s", address)
	return listenAndServe(address, newRouter())
}
