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
	Context_root string
	Server_listen string
	Server_port string
}

// The struct to store the configuration data
var config Config
// 12 hour caching that cleans up every 15 minutes
var cache_instance *cache.Cache
//Link to dropbox
var db *dropbox.Dropbox

var do_not_include []string

func main() {
	var err error

	config = Config{}
	cache_instance = cache.New(12*time.Hour, 15*time.Minute)
	db = dropbox.NewDropbox()

	do_not_include = []string{}
	do_not_include = append(do_not_include, ".txt")

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
    	//rest.Get("/", list_arch),
    	//rest.Get("/#arch", list_software),
    	//rest.Get("/#arch/#software", list_versions),
    	//rest.Get("/#arch/#software/#version", list_targets),
        rest.Get(strings.Join([]string{"/",config.Context_root,"/#arch/#software/#version/#target"},""), lookup_target),
    )
    if err != nil {
        log.Fatal(err)
    }
    api.SetApp(router)
    s := []string{}
    s = append(s, config.Server_listen)
    s = append(s, config.Server_port)
    server_listen := strings.Join(s, ":")
    log.Fatal(http.ListenAndServe(server_listen, api.MakeHandler()))
}

// func list_targets(w rest.ResponseWriter, r *rest.Request) {
// 	arch := r.PathParam("arch")
// 	software := r.PathParam("software")
// 	version := r.PathParam("version")
// 	w.WriteJson(map[string]string{"arch": arch, "software": software, "version": version})
// }

func lookup_target(w rest.ResponseWriter, r *rest.Request) {
	arch := r.PathParam("arch")
	software := r.PathParam("software")
	version := r.PathParam("version")
	//target := r.PathParam("target")

	latest := get_latest(arch,software,version)
	latest_link := get_link(cache_instance, db, latest.Path)

	w.Header().Set("Location", latest_link.URL)
  	w.WriteHeader(302)
    //w.WriteJson(map[string]string{"arch": arch, "software": software, "version": version, "target": target, "latest": latest.Path, "latest_link": latest_link.URL})
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
    		//fmt.Printf("Loaded from cache")
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
				include := true
				for _,lookup := range do_not_include {
					if strings.Contains(entry.Path, lookup) {
						include = false
					}
				}
				if include {
					paths = append(paths, entry)
				}
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

/*
	Use the arch, software and version to find the latest
*/
func get_latest(arch string, software string, version string) dropbox.Entry {
	var target_path string
	if version == "nightly" {
		target_path = arch
	} else {
		directories := get_directories(cache_instance, db, arch)
		mTime := time.Time(dropbox.DBTime{})
		var latest_directory dropbox.Entry	
		for _,dir := range directories {
			if strings.Contains(dir.Path, version) {
				if time.Time(dir.Modified).After(mTime) {
					mTime = time.Time(dir.Modified)
					latest_directory = dir
				}
			}
		}
		target_path = latest_directory.Path
	}

	s := []string{}
	s = append(s, software)
	s = append(s, "-")
	search := strings.Join(s,"")

	mTime := time.Time(dropbox.DBTime{})
	var latest_file dropbox.Entry
	files := get_files(cache_instance, db, target_path)
	for _,file := range files {
		if strings.Contains(file.Path, search) {
			if time.Time(file.Modified).After(mTime) {
				mTime = time.Time(file.Modified)
				latest_file = file
			}
		}
	}

	return latest_file
}