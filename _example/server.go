package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/demianbucik/sail"
	"github.com/demianbucik/sail/utils"
)

func main() {
	filePath := flag.String("env", "../env.yaml", "Path to YAML file with environment variables")
	assetsPath := flag.String("assets", ".", "Path to folder with HTML assets you'd like to serve")
	port := flag.Int("port", 8000, "Server port")

	flag.Parse()

	sail.Init(utils.GetParseFromYAMLFunc(*filePath))

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(*assetsPath))

	mux.HandleFunc("/send-email", sail.SendEmailHandler)
	mux.Handle("/", fs)

	fmt.Printf("Listening at http://localhost:%d\n", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), mux)
	fmt.Println(err)
}
