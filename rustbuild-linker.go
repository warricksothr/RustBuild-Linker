package main

import (
	"io/ioutil"
	"fmt"
	"log"
	//"net/http"
	"gopkg.in/yaml.v2"
	"github.com/stacktic/dropbox"
	//"github.com/ant0ine/go-json-rest/rest"
)

type Config struct {
	Api_secret string
	Api_key string
	Client_token string
	App_folder string
	Socket_file string
}

func main() {
	var err error
    var db *dropbox.Dropbox

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
    fmt.Printf("--- config_file dump:\n%s\n\n", data)
    fmt.Printf("--- config dump:\n%s\n\n", config)

	db = dropbox.NewDropbox()
	db.SetAppInfo(config.Api_key, config.Api_secret)
	if len(config.Client_token) >= 1 {
		db.SetAccessToken(config.Client_token)
	} else {
		if err = db.Auth(); err != nil {
	        fmt.Println(err)
	        return
	    }
	    config.Client_token = db.AccessToken()
	    db.SetAccessToken(config.Client_token)
	    d, marshal_err := yaml.Marshal(&config)
        if marshal_err != nil {
            log.Fatal("error: %v", marshal_err)
        }
        ioutil.WriteFile("config.yml",[]byte(d), 0644)
	}

	
}
