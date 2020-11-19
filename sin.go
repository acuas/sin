package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/acuas/sin/db/db"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const programURL = "localhost:8081"

type app struct {
	db *db.PasteDatabase
}

func main() {
	sin := &app{db.CreatePasteDatabase("sin.db")}
	err := http.ListenAndServe("127.0.0.1:8081", handler(sin))
	if err != nil {
		log.Fatal(err)
	}
}

func handler(app *app) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", app.home).
		Methods("GET")

	r.HandleFunc("/", app.postPaste).
		Methods("POST")

	r.HandleFunc("/robots.txt", app.robots).
		Methods("GET")

	r.HandleFunc("/h1dd3n", app.h1dd3n).
		Methods("GET")
	r.HandleFunc("/{pasteID}", app.retrievePaste)
	return r
}

func (sin *app) retrievePaste(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pasteID := vars["pasteID"]

	paste, err := sin.db.RetrievePaste(pasteID)
	if err != nil {
		// TODO: 404?
		log.Printf("%s\n", err)
		return
	}
	w.Write(paste.Data)
}

func (sin *app) postPaste(w http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024) // 2 Mb

	for i := 1; ; i++ {
		key := fmt.Sprintf("f:%d", i)
		contents := []byte(r.FormValue(key))

		if len(contents) == 0 {

			f, _, err := r.FormFile(key)
			if err != nil {
				break
			}

			contents, err = ioutil.ReadAll(f)
			if err != nil {
				break
			}

		}

		paste, err := sin.db.StorePaste(contents)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
			return
		}

		fmt.Fprintf(w, "https://%s/%s\n", programURL, paste.ID)

		log.Printf("Storing paste %s from %s\n", paste.ID, ip)
	}
}

func (sin *app) home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, Help)
}

var Help = fmt.Sprintf(`
sin(1)                               sin                                  sin(1)

NAME

	sin: command line pastebin.


TL;DR

	~$ echo Hello world. | curl -F 'f:1=<-' %[1]s
	https://%[1]s/fpW


GET

	%[1]s/ID
		raw


POST

	%[1]s/

		f:N    contents or attached file.

	where N is a unique number within request. (This allows you to post
	multiple files at once.)

	returns: https://%[1]s/id for N in request


EXAMPLES

	Anonymous, unnamed paste, two ways:

		cat file.ext | curl -F 'f:1=<-' %[1]s
		curl -F 'f:1=@file.ext' %[1]s
`, programURL)

func (sin *app) robots(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, RobotsTxt)
}

var RobotsTxt = `
User-agent: *
Disallow: /h1dd3n/
`

func (sin *app) h1dd3n(w http.ResponseWriter, r *http.Request) {
	ref := r.Referer()
	if ref != "p0st3b7n" {
		fmt.Fprintf(w, `Access disallowed. You are visiting from "" while authorized users should come only from a secret client!`)
	} else {
		fmt.Fprintf(w, fmt.Sprint(1))
	}
}
