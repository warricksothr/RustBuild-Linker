package main

import (
	"io/ioutil"
	"fmt"
	"log"
	//"net/http"
	"gopkg.in/yaml.v2"
	//"github.com/stacktic/dropbox"
	//"github.com/ant0ine/go-json-rest/rest"
)

type Config struct {
	api_secret string
	api_key string
	app_folder string
	socket_file string
}

func main() {
	// The struct to store the configuration data
	config := Config{}

	data, read_err := ioutil.ReadFile("config.yml")
	if read_err != nil {
		log.Fatal(read_err)
	}

	yaml_err := yaml.Unmarshal(data, &config)
	if yaml_err != nil {
    	log.Fatal("error: %v", yaml_err)
    }
    fmt.Printf("--- config dump:\n%s\n\n", string(data))
}
