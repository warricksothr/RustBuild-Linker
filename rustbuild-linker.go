package main

import (
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pmylund/go-cache"
	"github.com/stacktic/dropbox"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Api_secret    string
	Api_key       string
	Client_token  string
	Context_root  string
	Server_listen string
	Server_port   string
}

type Archive struct {
	Software string
	Date     string
	Version  string
	Tag      string
}

func (a Archive) Init(path string) *Archive {
	_, filename := filepath.Split(path)
	parts := strings.Split(filename, "-")
	a.Software = parts[0]
	a.Version = strings.Join([]string{parts[1], parts[2]}, " ")
	a.Date = strings.Join([]string{parts[3], parts[4], parts[5]}, "-")
	a.Tag = parts[6]
	return &a
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
		ioutil.WriteFile("config.yml", []byte(d), 0644)
	}

	// root_paths := get_directories(cache, db, "")
	// fmt.Printf("--- paths dump:\n%s\n\n", root_paths)

	// nightly_files := get_files(cache, db, "ARMv7")
	// fmt.Printf("--- paths dump:\n%s\n\n", nightly_files)

	// setup server to link
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	router, err := rest.MakeRouter(
		rest.Get(strings.Join([]string{config.Context_root, ""}, "/"), list_arch),
		rest.Get(strings.Join([]string{config.Context_root, "#arch"}, "/"), list_software),
		rest.Get(strings.Join([]string{config.Context_root, "#arch/#software"}, "/"), list_versions),
		rest.Get(strings.Join([]string{config.Context_root, "#arch/#software/#version"}, "/"), list_targets),
		rest.Get(strings.Join([]string{config.Context_root, "#arch/#software/#version/#target"}, "/"), link_target),
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

func list_arch(w rest.ResponseWriter, r *rest.Request) {
	// Use caching to reduce calls to the Dropbox API
	cache_path := "arches"
	data, found := cache_instance.Get(cache_path)
	if found {
		if cached, ok := data.([]string); ok {
			w.WriteJson(cached)
			return
		} else {
			log.Println("Error: Unable to retrieve from cache")
		}
	}

	arches := []string{}
	directories := get_directories(cache_instance, db, "/")
	for _, arch := range directories {
		arches = append(arches, strings.Replace(arch.Path, "/", "", -1))
	}
	cache_instance.Set(cache_path, arches, 0)
	w.WriteJson(arches)
}

func list_software(w rest.ResponseWriter, r *rest.Request) {
	arch := r.PathParam("arch")

	// Use caching to reduce calls to the Dropbox API
	cache_path := arch
	data, found := cache_instance.Get(cache_path)
	if found {
		if cached, ok := data.([]string); ok {
			w.WriteJson(cached)
			return
		} else {
			log.Println("Error: Unable to retrieve from cache")
		}
	}

	softwares := make(map[string]string)
	files := get_files(cache_instance, db, arch)
	for _, file := range files {
		archive := new(Archive)
		archive = archive.Init(file.Path)
		softwares[archive.Software] = ""
	}
	keys := make([]string, 0, len(softwares))
	for k := range softwares {
		keys = append(keys, k)
	}
	cache_instance.Set(cache_path, keys, 0)
	w.WriteJson(keys)
}

func list_versions(w rest.ResponseWriter, r *rest.Request) {
	w.WriteJson([]string{"nightly", "beta", "stable"})
}

func list_targets(w rest.ResponseWriter, r *rest.Request) {
	arch := r.PathParam("arch")
	software := r.PathParam("software")
	version := r.PathParam("version")

	// Doesn't need to be cached, as its calls are already cached.

	targets := []string{"latest"}
	target_path := get_target_path(arch, software, version)
	files := get_files(cache_instance, db, target_path)
	for _, file := range files {
		archive := new(Archive)
		archive = archive.Init(file.Path)
		targets = append(targets, archive.Tag)
	}
	w.WriteJson(targets)
}

func link_target(w rest.ResponseWriter, r *rest.Request) {
	arch := r.PathParam("arch")
	software := r.PathParam("software")
	version := r.PathParam("version")
	target := r.PathParam("target")

	target_file, found := get_target(arch, software, version, target)
	if found {
		target_link := get_link(cache_instance, db, target_file.Path)
		w.Header().Set("Location", target_link)
		w.WriteHeader(302)
	} else {
		w.WriteJson(map[string]string{"error": "Target Not Found"})
	}
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

/*
	Actually get a list of directories or files from the dropbox connection
*/
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
	cache_path := strings.Join(s, "")

	data, found := cache.Get(cache_path)
	if found {
		if cached_paths, ok := data.([]dropbox.Entry); ok {
			return (cached_paths)
		} else {
			log.Println("Error: Unable to retrieve from cache")
		}
	}

	entry, err := db.Metadata(path, true, false, "", "", 500)
	if err != nil {
		log.Println(err)
		return []dropbox.Entry{}
	}
	paths := make([]dropbox.Entry, 0)
	for i := 0; i < len(entry.Contents); i++ {
		entry := entry.Contents[i]
		if directories {
			if entry.IsDir {
				paths = append(paths, entry)
			}
		} else {
			if !entry.IsDir {
				include := true
				for _, lookup := range do_not_include {
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
	Divine the correct target path from the provided info
*/
func get_target_path(arch string, software string, version string) string {
	var target_path string
	if version == "nightly" {
		target_path = arch
	} else {
		directories := get_directories(cache_instance, db, arch)
		mTime := time.Time(dropbox.DBTime{})
		var latest_directory dropbox.Entry
		for _, dir := range directories {
			if strings.Contains(dir.Path, version) {
				if time.Time(dir.Modified).After(mTime) {
					mTime = time.Time(dir.Modified)
					latest_directory = dir
				}
			}
		}
		target_path = latest_directory.Path
	}
	return target_path
}

/*
	Returns a shared link to dropbox file
*/
func get_link(cache *cache.Cache, db *dropbox.Dropbox, path string) string {

	// Use caching to reduce calls to the Dropbox API
	cache_path := strings.Join([]string{"link", path}, ":")
	data, found := cache.Get(cache_path)
	if found {
		if cached, ok := data.(string); ok {
			return cached
		} else {
			log.Println("Error: Unable to retrieve from cache")
		}
	}

	link, err := db.Shares(path, false)
	if err != nil {
		log.Println(err)
		return ""
	}
	cache.Set(cache_path, link.URL, 0)
	return link.URL
}

func get_target(arch string, software string, version string, target string) (dropbox.Entry, bool) {
	if target == "latest" {
		return get_latest(arch, software, version), true
	} else {
		target_path := get_target_path(arch, software, version)
		files := get_files(cache_instance, db, target_path)
		for _, file := range files {
			archive := new(Archive)
			archive = archive.Init(file.Path)
			if archive.Software == software {
				if archive.Tag == target {
					return file, true
				}
			}
		}
	}
	return dropbox.Entry{}, false
}

/*
	Use the arch, software and version to find the latest
*/
func get_latest(arch string, software string, version string) dropbox.Entry {
	target_path := get_target_path(arch, software, version)

	s := []string{}
	s = append(s, software)
	s = append(s, "-")
	search := strings.Join(s, "")

	mTime := time.Time(dropbox.DBTime{})
	var latest_file dropbox.Entry
	files := get_files(cache_instance, db, target_path)
	for _, file := range files {
		if strings.Contains(file.Path, search) {
			if time.Time(file.Modified).After(mTime) {
				mTime = time.Time(file.Modified)
				latest_file = file
			}
		}
	}

	return latest_file
}
