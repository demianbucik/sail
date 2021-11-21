package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"

	"github.com/demianbucik/sail"
	"github.com/demianbucik/sail/config"
)

func main() {
	envFilePath := flag.String("env", "../env.yaml", "Path to YAML file with environment variables")
	assetsPath := flag.String("assets", ".", "Path to folder with HTML assets you'd like to serve")
	port := flag.Int("port", 8000, "Server port")

	flag.Parse()

	sail.Init(config.GetParseFromYAMLFunc(*envFilePath))
	log.SetHandler(text.Default)
	log.SetLevel(log.DebugLevel)

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(*assetsPath))

	mux.HandleFunc("/send-email", sail.SendEmailHandler)
	mux.Handle("/", fs)

	log.Infof("Listening at http://localhost:%d", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), mux)
	log.Infof("%s", err)
}
