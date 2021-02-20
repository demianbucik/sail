package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/demianbucik/sail"
	"github.com/demianbucik/sail/utils"
)

func main() {
	filePath := flag.String("env", "../env.yaml", "Path to YAML file with environment variables")
	flag.Parse()

	sail.Init(utils.GetParseFromYAMLFunc(*filePath))

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("."))

	mux.HandleFunc("/send-email", sail.SendEmailHandler)
	mux.Handle("/", fs)

	err := http.ListenAndServe(":8000", mux)
	log.Println(err)
}
