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

	log.Println("Starting server on :3000")
	//Print statement
	err := http.ListenAndServe(":3000", mux)
	log.Fatal(err)
}

func Welcome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome, we have what you need!!")
	fmt.Fprintf(w, "Coffe for everyone")
	fmt.Fprintf(w, "Coffe for everyone")

}
