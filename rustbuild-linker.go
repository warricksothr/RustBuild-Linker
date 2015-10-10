package main

import (
	"io/ioutil"
	"strings"
	"fmt"
	"log"
    "time"
	"net/http"
	"gopkg.in/yaml.v2"
	"github.com/pmylund/go-cache"
	"github.com/stacktic/dropbox"
	"github.com/ant0ine/go-json-rest/rest"
)

type Config struct {
	Api_secret string
	Api_key string
	Client_token string
	App_folder string
	Socket_file string
}

// The struct to store the configuration data
var config Config
// 12 hour caching that cleans up every 15 minutes
var c *cache.Cache
//Link to dropbox
var db *dropbox.Dropbox

func main() {
	var err error

	config = Config{}
	c = cache.New(12*time.Hour, 15*time.Minute)
	db = dropbox.NewDropbox()

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

	// root_paths := get_directories(cache, db, "")
	// fmt.Printf("--- paths dump:\n%s\n\n", root_paths)

	// nightly_files := get_files(cache, db, "ARMv7")
	// fmt.Printf("--- paths dump:\n%s\n\n", nightly_files)

	// setup server to link
	api := rest.NewApi()
    api.Use(rest.DefaultDevStack...)
    router, err := rest.MakeRouter(
        rest.Get("/#arch/#software/#version/#target", lookup_target),
    )
    if err != nil {
        log.Fatal(err)
    }
    api.SetApp(router)
    log.Fatal(http.ListenAndServe(":8080", api.MakeHandler()))
}

func lookup_target(w rest.ResponseWriter, req *rest.Request) {
	arch := req.PathParam("arch")
	software := req.PathParam("software")
	version := req.PathParam("version")
	target := req.PathParam("target")
    w.WriteJson(map[string]string{"arch": arch, "software": software, "version": version, "target": target})
}

/*
	Get only a slice of the directories at a path
*/
func get_directories(cache *cache.Cache, db *dropbox.Dropbox, path string) []dropbox.Entry {
	return get(cache, db, path, true)
}

/*
	Get only a slice of the directories at a path
*/
func get_files(cache *cache.Cache, db *dropbox.Dropbox, path string) []dropbox.Entry {
	return get(cache, db, path, false)
}

func get(cache *cache.Cache, db *dropbox.Dropbox, path string, directories bool) []dropbox.Entry {
	// Use caching to reduce calls to the Dropbox API
	var cache_descriptor string 
	if directories {
		cache_descriptor = "dirs:"
	} else {
		cache_descriptor = "files:"
	}
	s := []string{}
	s = append(s, cache_descriptor)
	s = append(s, path)
	cache_path := strings.Join(s,"")

	data, found := cache.Get(cache_path)
    if found {
    	if cached_paths, ok := data.([]dropbox.Entry); ok {
    		fmt.Printf("Loaded from cache")
		    return (cached_paths)
		} else {
			log.Fatal("Unable to retrieve from cache")
		}   
    }

	entry, err := db.Metadata(path,true,false,"","",500);
	if err != nil {
		log.Fatal(err)
	}
	paths := make([]dropbox.Entry, 0)
	for i := 0; i < len(entry.Contents); i++ {
		entry := entry.Contents[i]
		if (directories) {
			if entry.IsDir {
				paths = append(paths, entry)
			}
		} else {
			if ! entry.IsDir {
				paths = append(paths, entry)
			}
		}
	}
	cache.Set(cache_path, paths, 0)
	return paths
}

/*
	Returns a shared link to dropbox file
*/
func get_link(cache *cache.Cache, db *dropbox.Dropbox, path string) *dropbox.Link {
	link, err := db.Shares(path, false)
	if err != nil {
		log.Fatal(err)
	}
	return link
}