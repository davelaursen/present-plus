// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !appengine

package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
var plusDirPath string

func main() {
	httpAddr := flag.String("http", "127.0.0.1:4999", "HTTP service address (e.g., '127.0.0.1:4999')")
	originHost := flag.String("orighost", "", "host component of web origin URL (e.g., 'localhost')")
	flag.StringVar(&basePath, "base", "", "base path for slide template and static resources")
	flag.BoolVar(&present.PlayEnabled, "play", true, "enable playground (permit execution of arbitrary user code)")
	nativeClient := flag.Bool("nacl", false, "use Native Client environment playground (prevents non-Go code execution)")
	flag.StringVar(&defaultTheme, "theme", "", "the default theme to apply when no custom styles are defined")
	flag.StringVar(&repoPath, "repo", "", "path for theme repository")
	flag.Parse()

	if repoPath != "" {
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
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

	if basePath == "" {
		basePath = p.Dir

		tmpDir := filepath.Join(basePath, "static", "tmp")
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't remove tmp directory '%s': %v\n", tmpDir, err)
			os.Exit(1)
		}
	}
	plusDirPath = getPlusDirPath()
	if repoPath == "" {
		repoPath, _ = filepath.Abs(filepath.Join(plusDirPath, "themes"))
	}

	args := os.Args[1:]
	if len(args) > 0 && args[0][0:1] != "-" {
		switch args[0] {
		case "install":
			installTheme(args)
		case "uninstall":
			uninstallTheme(args)
		default:
			fmt.Fprintf(os.Stderr, "'%s' is not a valid command\n", args[0])
			os.Exit(1)
		}

		os.Exit(0)
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

func getPlusDirPath() string {
	usr, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get current user: %s\n", err)
		os.Exit(1)
	}

	var plusDirPath string
	if runtime.GOOS == "windows" {
		plusDirPath = filepath.Join(usr.HomeDir, "present_plus")
	} else {
		plusDirPath = filepath.Join(usr.HomeDir, ".present_plus")
	}
	themesDir := filepath.Join(plusDirPath, "themes")
	if _, err = os.Stat(themesDir); os.IsNotExist(err) {
		err := os.MkdirAll(themesDir, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory '%s': %s\n", themesDir, err)
			os.Exit(1)
		}
	}
	return plusDirPath
}

func installTheme(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Invalid use of '%s' command\n", args[0])
		os.Exit(1)
	}

	tmpDir, _ := filepath.Abs(filepath.Join(plusDirPath, "tmp"))
	os.Mkdir(tmpDir, 0777)
	rmvTmpDir := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't remove tmp directory '%s': %v\n", tmpDir, err)
			os.Exit(1)
		}
	}

	parts := strings.Split(args[1], "/")
	for len(parts) > 0 && (strings.HasPrefix(parts[0], "http") || parts[0] == "") {
		parts = parts[1:]
	}
	if len(parts) <= 2 || strings.ToLower(parts[0]) != "github.com" {
		fmt.Fprintf(os.Stderr, "'%s' is not a valid GitHub repo\n", args[1])
		rmvTmpDir()
		os.Exit(1)
	}
	gitRepo := "https://" + filepath.Join(parts[0], parts[1], parts[2]+".git")
	cmd := exec.Command("git", "clone", gitRepo)
	cmd.Dir = tmpDir
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cloning repo '%s': %s\n", gitRepo, err)
		rmvTmpDir()
		os.Exit(1)
	}
	srcPath := filepath.Join(tmpDir, strings.Join(parts[2:], "/"))
	destPath := filepath.Join(plusDirPath, "themes", parts[len(parts)-1])
	if err := os.Rename(srcPath, destPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying theme to themes folder: %s\n", err)
		rmvTmpDir()
		os.Exit(1)
	}
	rmvTmpDir()
}

func uninstallTheme(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Invalid use of '%s' command\n", args[0])
		os.Exit(1)
	}

	themeDir, _ := filepath.Abs(filepath.Join(plusDirPath, "themes", args[1]))
	if err := os.RemoveAll(themeDir); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't remove theme directory '%s': %v\n", themeDir, err)
		os.Exit(1)
	}
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
