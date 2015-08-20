// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !appengine

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/davelaursen/present-plus/present"
	"golang.org/x/tools/playground/socket"
)

const basePkg = "github.com/davelaursen/present-plus"

var basePath string
var repoPath string
var defaultTheme string

func main() {
	type Config struct {
		HTTP         string `json:"http"`
		OrigHost     string `json:"orighost"`
		Base         string `json:"base"`
		Play         bool   `json:"play"`
		NativeClient bool   `json:"nacl"`
		Theme        string `json:"theme"`
		Repo         string `json:"repo"`
	}
	// set config instance with default values
	config := Config{
		HTTP:         "127.0.0.1:4999",
		OrigHost:     "",
		Base:         "",
		Play:         true,
		NativeClient: false,
		Theme:        "",
		Repo:         "",
	}

	// ensure config file exists
	usr, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get current user: %s\n", err)
		fmt.Fprintf(os.Stderr, "Using default settings\n")
	} else {
		var configFilePath string
		if runtime.GOOS == "windows" {
			configFilePath = filepath.Join(usr.HomeDir, "ppconfig.json")
		} else {
			configFilePath = filepath.Join(usr.HomeDir, ".ppconfig")
		}
		_, err = os.Stat(configFilePath)
		if os.IsNotExist(err) {
			f, err := os.Create(configFilePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating file '%s': %s\n", configFilePath, err)
				os.Exit(1)
			}
			defer f.Close()

			f.Write([]byte("{}"))
			f.Close()
		}

		// parse config file
		configFile, err := os.Open(configFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open config file '%s': %s\n", configFilePath, err)
			os.Exit(1)
		}

		// override defaults with config file values
		jsonParser := json.NewDecoder(configFile)
		if err = jsonParser.Decode(&config); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse config file '%s': %s\n", configFilePath, err)
			os.Exit(1)
		}
	}

	// override defaults & config file values with command line values
	httpAddr := flag.String("http", config.HTTP, "HTTP service address (e.g., '127.0.0.1:4999')")
	originHost := flag.String("orighost", config.OrigHost, "host component of web origin URL (e.g., 'localhost')")
	flag.StringVar(&basePath, "base", config.Base, "base path for slide template and static resources")
	flag.BoolVar(&present.PlayEnabled, "play", config.Play, "enable playground (permit execution of arbitrary user code)")
	nativeClient := flag.Bool("nacl", config.NativeClient, "use Native Client environment playground (prevents non-Go code execution)")
	flag.StringVar(&defaultTheme, "theme", config.Theme, "the default theme to apply when no custom styles are defined")
	flag.StringVar(&repoPath, "repo", config.Repo, "path for theme repository")
	flag.Parse()

	if repoPath != "" {
		_, err = os.Stat(repoPath)
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Repo directory '%s' does not exist\n", repoPath)
			os.Exit(1)
		}
	}

	p, err := build.Default.Import(basePkg, "", build.FindOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't find gopresent files: %v\n", err)
		fmt.Fprintf(os.Stderr, basePathMessage, basePkg)
		os.Exit(1)
	}
	defaultBasePath := p.Dir

	if basePath == "" {
		basePath = defaultBasePath

		tmpDir := filepath.Join(basePath, "static", "tmp")
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't remove tmp directory '%s': %v\n", tmpDir, err)
			os.Exit(1)
		}
	}
	if repoPath == "" {
		themeRepo, _ := filepath.Abs(filepath.Join(defaultBasePath, "..", "present-plus-themes"))
		_, err = os.Stat(themeRepo)
		if !os.IsNotExist(err) {
			repoPath = themeRepo
		}
	}

	err = initTemplates(basePath)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	ln, err := net.Listen("tcp", *httpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	origin := &url.URL{Scheme: "http"}
	if *originHost != "" {
		origin.Host = net.JoinHostPort(*originHost, port)
	} else if ln.Addr().(*net.TCPAddr).IP.IsUnspecified() {
		name, _ := os.Hostname()
		origin.Host = net.JoinHostPort(name, port)
	} else {
		reqHost, reqPort, err := net.SplitHostPort(*httpAddr)
		if err != nil {
			log.Fatal(err)
		}
		if reqPort == "0" {
			origin.Host = net.JoinHostPort(reqHost, port)
		} else {
			origin.Host = *httpAddr
		}
	}

	if present.PlayEnabled {
		if *nativeClient {
			socket.RunScripts = false
			socket.Environ = func() []string {
				if runtime.GOARCH == "amd64" {
					return environ("GOOS=nacl", "GOARCH=amd64p32")
				}
				return environ("GOOS=nacl")
			}
		}
		playScript(basePath, "SocketTransport")
		http.Handle("/socket", socket.NewHandler(origin))
	}
	http.Handle("/static/", http.FileServer(http.Dir(basePath)))

	if !ln.Addr().(*net.TCPAddr).IP.IsLoopback() &&
		present.PlayEnabled && !*nativeClient {
		log.Print(localhostWarning)
	}

	log.Printf("Open your web browser and visit %s", origin.String())
	log.Fatal(http.Serve(ln, nil))
}

func playable(c present.Code) bool {
	return present.PlayEnabled && c.Play
}

func environ(vars ...string) []string {
	env := os.Environ()
	for _, r := range vars {
		k := strings.SplitAfter(r, "=")[0]
		var found bool
		for i, v := range env {
			if strings.HasPrefix(v, k) {
				env[i] = r
				found = true
			}
		}
		if !found {
			env = append(env, r)
		}
	}
	return env
}

const basePathMessage = `
By default, gopresent locates the slide template files and associated
static content by looking for a %q package
in your Go workspaces (GOPATH).

You may use the -base flag to specify an alternate location.
`

const localhostWarning = `
WARNING!  WARNING!  WARNING!

The present server appears to be listening on an address that is not localhost.
Anyone with access to this address and port will have access to this machine as
the user running present.

To avoid this message, listen on localhost or run with -play=false.

If you don't understand this message, hit Control-C to terminate this process.

WARNING!  WARNING!  WARNING!
`
