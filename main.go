package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {

	mux := http.NewServeMux()
	// creating a object for handler func

	mux.HandleFunc("/", Welcome)
	//Creating a handler function and Welcome is an endpoint

	log.Println("Starting server on :4000")
	//Print statement
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}

func Welcome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome, we have what you need!!/n")
	fmt.Fprintf(w, "Coffe for everyone")
}
