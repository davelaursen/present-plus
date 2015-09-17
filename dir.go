// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/davelaursen/present-plus/present"
)

func init() {
	http.HandleFunc("/", dirHandler)
}

var tmpIndex = 0
var mutex = &sync.Mutex{}

// dirHandler serves a directory listing for the requested path, rooted at basePath.
func dirHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		http.Error(w, "not found", 404)
		return
	}
	const base = "."
	name := filepath.Join(base, r.URL.Path)
	if isDoc(name) {
		err := renderDoc(w, name)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
		}
		return
	}
	if isDir, err := dirList(w, name); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	} else if isDir {
		return
	}
	http.FileServer(http.Dir(base)).ServeHTTP(w, r)
}

func isDoc(path string) bool {
	_, ok := contentTemplate[filepath.Ext(path)]
	return ok
}

var (
	// dirListTemplate holds the front page template.
	dirListTemplate *template.Template

	// contentTemplate maps the presentable file extensions to the
	// template to be executed.
	contentTemplate map[string]*template.Template
)

func initTemplates(base string) error {
	// Locate the template file.
	actionTmpl := filepath.Join(base, "templates/action.tmpl")

	contentTemplate = make(map[string]*template.Template)

	for ext, contentTmpl := range map[string]string{
		".slide":   "slides.tmpl",
		".article": "article.tmpl",
	} {
		contentTmpl = filepath.Join(base, "templates", contentTmpl)

		// Read and parse the input.
		tmpl := present.Template()
		tmpl = tmpl.Funcs(template.FuncMap{"playable": playable})
		if _, err := tmpl.ParseFiles(actionTmpl, contentTmpl); err != nil {
			return err
		}
		contentTemplate[ext] = tmpl
	}

	var err error
	dirListTemplate, err = template.ParseFiles(filepath.Join(base, "templates/dir.tmpl"))
	if err != nil {
		return err
	}

	return nil
}

// renderDoc reads the present file, gets its template representation,
// and executes the template, sending output to w.
func renderDoc(w io.Writer, docFile string) error {
	// Read the input and build the doc structure.
	doc, err := parse(docFile, 0)
	if err != nil {
		return err
	}
	ext := filepath.Ext(docFile)
	if doc.Theme == "" && defaultTheme != "" &&
		((ext == ".article" && len(doc.ArticleStylesheets) == 0) || (ext == ".slide" && len(doc.SlideStylesheets) == 0)) {
		doc.Theme = defaultTheme
	}
	if doc.Theme != "" {
		parseTheme(docFile, doc)
	}

	// Find which template should be executed.
	tmpl := contentTemplate[filepath.Ext(docFile)]

	// Execute the template.
	return doc.Render(w, tmpl)
}

func parse(name string, mode present.ParseMode) (*present.Doc, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return present.Parse(f, name, 0)
}

func isDir(path string) bool {
	src, err := os.Stat(path)
	if err == nil && src != nil {
		return src.IsDir()
	}
	return false
}

func loadTheme(dirPath, themeName string) (Theme, string, bool) {
	dirPath, err := filepath.Abs(dirPath)
	// find theme folder
	lookedIn := ""
	themePath := ""
	// first look for plus-themes folder in current or ancestor directory
	prevPath := ""
	for dirPath != prevPath {
		path := filepath.Join(dirPath, "plus-themes")
		lookedIn += "\n  " + path
		if isDir(path) {
			path = filepath.Join(path, themeName)
			if isDir(path) {
				themePath = path
				break
			}
		}
		prevPath = dirPath
		dirPath = filepath.Dir(dirPath)
	}
	// if not found, look in theme repo
	if themePath == "" {
		path := filepath.Join(repoPath, themeName)
		lookedIn += "\n  " + repoPath
		if isDir(path) {
			themePath = path
		}
	}
	// if not found, look in the default theme folder
	if themePath == "" {
		themePath = filepath.Join(basePath, "static", "themes", themeName)
		lookedIn += "\n  " + filepath.Dir(themePath)
		if !isDir(themePath) {
			log.Printf("Theme folder '%s' could not be found at any of the following locations:%s\n", themeName, lookedIn)
			return Theme{}, "", false
		}
	}

	// create tmp directory
	mutex.Lock()
	tmpDir := filepath.Join("static", "tmp", strconv.Itoa(tmpIndex))
	targetDir := filepath.Join(basePath, tmpDir)
	tmpIndex++
	mutex.Unlock()

	if err := os.MkdirAll(targetDir, 0777); err != nil {
		log.Printf("Error creating tmp directory: %v\n", err)
		return Theme{}, "", false
	}

	// copy files from theme folder into tmp directory
	dir, err := os.Open(themePath)
	if err != nil {
		log.Printf("Error opening theme directory: %v\n", err)
		return Theme{}, "", false
	}
	defer dir.Close()

	items, err := dir.Readdir(-1)
	for _, item := range items {
		source := filepath.Join(themePath, item.Name())
		target := filepath.Join(targetDir, item.Name())
		if err := copyFile(source, target); err != nil {
			log.Printf("Error copying theme file '%s' to tmp directory: %v\n", item.Name(), err)
		}
	}

	// read info from theme file and populate Theme object
	themeFile := filepath.Join(themePath, "theme.json")
	f, err := os.Open(themeFile)
	if err != nil {
		log.Printf("Error opening theme file: %v\n", err)
		return Theme{}, "", false
	}
	defer f.Close()
	var theme Theme

	jsonParser := json.NewDecoder(f)
	if err = jsonParser.Decode(&theme); err != nil {
		log.Printf("Error parsing JSON object from theme file: %v\n", err)
		return Theme{}, "", false
	}
	return theme, tmpDir, true
}

func parseTheme(name string, doc *present.Doc) {
	// first look for plus-themes folder in current or ancestor directory
	dirPath, err := filepath.Abs(filepath.Dir(name))
	if err != nil {
		log.Printf("Error attempting to determine absolute path of file '%s': %v\n", name, err)
		dirPath = "."
	}

	theme, tmpDir, success := loadTheme(dirPath, doc.Theme)
	if !success {
		return
	}

	if theme.ArticleStylesheets != nil {
		tempArr := []string{}
		for _, stylesheet := range theme.ArticleStylesheets {
			if stylesheet[0] != '/' {
				stylesheet = "/" + filepath.Join(tmpDir, stylesheet)
			}
			tempArr = append(tempArr, stylesheet)
		}
		doc.ArticleStylesheets = append(tempArr, doc.ArticleStylesheets...)
	}
	if theme.SlideStylesheets != nil {
		tempArr := []string{}
		for _, stylesheet := range theme.SlideStylesheets {
			if stylesheet[0] != '/' {
				stylesheet = "/" + filepath.Join(tmpDir, stylesheet)
			}
			tempArr = append(tempArr, stylesheet)
		}
		doc.SlideStylesheets = append(tempArr, doc.SlideStylesheets...)
	}
	if doc.HideLastSlide == "" {
		if theme.HideLastSlide != "" {
			doc.HideLastSlide = theme.HideLastSlide
		} else {
			doc.HideLastSlide = "false"
		}
	}
	if doc.ClosingMessage == "" {
		if theme.ClosingMessage != "" {
			doc.ClosingMessage = theme.ClosingMessage
		} else {
			doc.ClosingMessage = "Thank You"
		}
	}
}

func copyFile(source, dest string) error {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}
	}

	return err
}

// dirList scans the given path and writes a directory listing to w.
// It parses the first part of each .slide file it encounters to display the
// presentation title in the listing.
// If the given path is not a directory, it returns (isDir == false, err == nil)
// and writes nothing to w.
func dirList(w io.Writer, name string) (isDir bool, err error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	if isDir = fi.IsDir(); !isDir {
		return false, nil
	}
	fis, err := f.Readdir(0)
	if err != nil {
		return false, err
	}
	themeName := defaultTheme
	hidePath := false
	hideFileName := false
	d := &dirListData{Path: name, Title: "Go Talks"}
	for _, fi := range fis {
		// skip the golang.org directory
		if name == "." && fi.Name() == "golang.org" {
			continue
		}
		e := dirEntry{
			Name:         fi.Name(),
			Path:         filepath.ToSlash(filepath.Join(name, fi.Name())),
			ShowFileName: true,
		}
		if e.Name == "plus-config.json" {
			f2, err2 := os.Open(e.Path)
			if err2 != nil {
				log.Printf("Error opening directory config file: %v\n", err)
				continue
			}
			defer f2.Close()

			type DirConfig struct {
				Title        string `json:"title"`
				Theme        string `json:"theme"`
				HidePath     bool   `json:"hidePath"`
				HideFileName bool   `json:"hideFileName"`
			}
			var config DirConfig

			jsonParser := json.NewDecoder(f2)
			if err = jsonParser.Decode(&config); err != nil {
				log.Printf("Error parsing JSON object from directory config file: %v\n", err)
				continue
			}
			if config.Title != "" {
				d.Title = config.Title
			}
			if config.Theme != "" {
				themeName = config.Theme
			}
			hidePath = config.HidePath
			hideFileName = config.HideFileName
			continue
		}
		if fi.IsDir() && showDir(e.Name) {
			d.Dirs = append(d.Dirs, e)
			continue
		}
		if isDoc(e.Name) {
			if p, err := parse(e.Path, present.TitlesOnly); err != nil {
				log.Println(err)
			} else {
				e.Title = p.Title
			}
			switch filepath.Ext(e.Path) {
			case ".article":
				d.Articles = append(d.Articles, e)
			case ".slide":
				d.Slides = append(d.Slides, e)
			}
		} else if showFile(e.Name) {
			d.Other = append(d.Other, e)
		}
	}
	if themeName != "" {
		theme, tmpDir, success := loadTheme(name, themeName)
		if success && theme.DirectoryStylesheets != nil {
			tempArr := []string{}
			for _, stylesheet := range theme.DirectoryStylesheets {
				if stylesheet[0] != '/' {
					stylesheet = "/" + filepath.Join(tmpDir, stylesheet)
				}
				tempArr = append(tempArr, stylesheet)
			}
			d.Stylesheets = append(tempArr, d.Stylesheets...)
		}
	}

	if hidePath || d.Path == "." {
		d.Path = ""
	}

	if hideFileName {
		for i := range d.Slides {
			d.Slides[i].ShowFileName = false
		}
		for i := range d.Articles {
			d.Articles[i].ShowFileName = false
		}
	}
	sort.Sort(d.Dirs)
	sort.Sort(d.Slides)
	sort.Sort(d.Articles)
	sort.Sort(d.Other)
	return true, dirListTemplate.Execute(w, d)
}

// showFile reports whether the given file should be displayed in the list.
func showFile(n string) bool {
	switch filepath.Ext(n) {
	case ".pdf":
	case ".html":
	case ".go":
	default:
		return isDoc(n)
	}
	return true
}

// showDir reports whether the given directory should be displayed in the list.
func showDir(n string) bool {
	if len(n) > 0 && (n[0] == '.' || n[0] == '_') || n == "present" || n == "plus-themes" {
		return false
	}
	return true
}

type dirListData struct {
	Title                         string
	Stylesheets                   []string
	Path                          string
	Dirs, Slides, Articles, Other dirEntrySlice
}

type dirEntry struct {
	Name, Path, Title string
	ShowFileName      bool
}

type dirEntrySlice []dirEntry

func (s dirEntrySlice) Len() int           { return len(s) }
func (s dirEntrySlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s dirEntrySlice) Less(i, j int) bool { return s[i].Name < s[j].Name }

type Theme struct {
	DirectoryStylesheets []string `json:"directory-stylesheets"`
	ArticleStylesheets   []string `json:"article-stylesheets"`
	SlideStylesheets     []string `json:"slide-stylesheets"`
	HideLastSlide        string   `json:"hide-last-slide"`
	ClosingMessage       string   `json:"closing-message"`
}
