package main

import (
	"log"
	"net/http"
	"os"
)

var (
	CustomString      = "the Sous Demo App"
	Version, Revision string
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile | log.Ltime)
	log.Print("Booting...")
	host := os.Getenv("TASK_HOST")
	port := os.Getenv("PORT0")

	http.HandleFunc("/", func(w http.ResponseWriter, rq *http.Request) {
		log.Printf("Handling request: %v", rq)
		w.Write([]byte("HELLO WHIRLED!"))
	})

	log.Printf("Starting up - serving on %s:%s", host, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
