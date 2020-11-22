package main

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/acuas/sin/db"
	php "github.com/deuill/go-php"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

const programURL = "localhost:8081"

var adminHash string
var adminPassword string

type app struct {
	db *db.PasteDatabase
}

func main() {
	random()
	sin := &app{db.CreatePasteDatabase("sin")}
	err := http.ListenAndServe("0.0.0.0:8081", handler(sin))
	if err != nil {
		log.Fatal(err)
	}
}

var engine *php.Engine

func handler(app *app) http.Handler {
	r := mux.NewRouter()

	engine, _ = php.New()

	r.HandleFunc("/", app.home).
		Methods("GET")

	r.HandleFunc("/admin", app.admin).
		Methods("GET")

	r.HandleFunc("/getImage", app.getImage).
		Methods("GET")

	r.HandleFunc("/submit", app.postPaste).
		Methods("POST")

	r.HandleFunc("/console", app.console).
		Methods("GET")

	r.HandleFunc("/robots.txt", app.robots).
		Methods("GET")

	r.HandleFunc("/h1dd3n", app.h1dd3n).
		Methods("GET")
	r.HandleFunc("/paste", app.retrievePaste).Queries("id", "{.*}")

	r.HandleFunc("/{path}", app.index).
		Methods("GET")
	return r
}

func (sin *app) admin(w http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")
	errString := fmt.Sprintf(`Basic realm="Only username admin can access this page, with the correct password 
	(which has the MD5 hash %s). All passwords consist of 5 lowercase letters, followed by 4 digits."`, adminHash)

	if auth == "" {
		w.Header().Set("WWW-Authenticate", errString)
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		parts := strings.Split(strings.Trim(auth, " "), " ")

		if len(parts) != 2 {
			w.Header().Set("WWW-Authenticate", errString)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		method := parts[0]
		if method != "Basic" {
			w.Header().Set("WWW-Authenticate", errString)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		bs, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			w.Header().Set("WWW-Authenticate", errString)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		authParts := strings.Split(string(bs), ":")
		if len(authParts) != 2 {
			w.Header().Set("WWW-Authenticate", errString)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		username := authParts[0]
		if username != "admin" {
			w.Header().Set("WWW-Authenticate", errString)
			w.WriteHeader(http.StatusUnauthorized)

			return
		} else {
			pass := authParts[1]

			if pass == adminPassword {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Content-Type", "text/plain")

				w.Write([]byte("Congrats! You pwned us!"))
			} else {
				w.Header().Set("WWW-Authenticate", errString)
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	}
}

func random() {
	rand.Seed(time.Now().UTC().UnixNano())
	letters := "abcdefghijklmnopqrstuvwxyz"
	numbers := "0123456789"

	adminPassword = fmt.Sprintf("%c%c%c%c%c%c%c%c%c",
		letters[rand.Intn(len(letters))],
		letters[rand.Intn(len(letters))],
		letters[rand.Intn(len(letters))],
		letters[rand.Intn(len(letters))],
		letters[rand.Intn(len(letters))],
		numbers[rand.Intn(len(numbers))],
		numbers[rand.Intn(len(numbers))],
		numbers[rand.Intn(len(numbers))],
		numbers[rand.Intn(len(numbers))])

	adminHash = fmt.Sprintf("%x", md5.Sum([]byte(adminPassword)))
}

func (sin *app) getImage(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "image/png")

	if strings.Contains(req.URL.String(), "..") {
		// do not allow parent directory traversal
		w.WriteHeader(http.StatusBadRequest)
	} else {
		q := req.URL.Query()
		path := fmt.Sprintf("./joke/%s", q.Get("filename"))

		http.ServeFile(w, req, path)
	}
}

func (sin *app) index(w http.ResponseWriter, r *http.Request) {
	var outputPhp strings.Builder
	vars := mux.Vars(r)
	path := vars["path"]
	context, _ := engine.NewContext()
	context.Header = r.Header
	context.Output = &outputPhp
	err := context.Exec(path)
	if err != nil {
		w.WriteHeader(404)
	} else {
		if strings.Contains(outputPhp.String(), "Warning: Unknown: failed to open stream: No such file or directory in Unknown on line 0") {
			w.WriteHeader(404)
			fmt.Fprint(w, "404 not found")
		} else {
			fmt.Fprint(w, outputPhp.String())
		}
	}
}

func (sin *app) retrievePaste(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	pasteID := r.FormValue("id")
	paste, err := sin.db.RetrievePaste(pasteID)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	fmt.Fprintf(w, `<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title>PasteIt!</title>
		<style>
			h1 {
				text-align: center;
			}
			#notebox {
				display: block;
				margin-left: auto;
				margin-right: auto;
				resize: none;
			}
		</style>
	</head>
	<body>
		<h1>Here's your paste!</h1>
		<textarea id="notebox" rows="25" cols="100">%s</textarea>
	</body>
	</html>`, paste.Data)
}

func (sin *app) postPaste(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024) // 2 Mb
	contents, _ := ioutil.ReadAll(r.Body)
	paste, err := sin.db.StorePaste(contents)
	if err != nil {
		fmt.Fprintf(w, "%s", err)
		return
	}
	w.Write([]byte(string(paste.ID)))
}

func clientIPAddr(req *http.Request) string {
	ipaddr := req.Header.Get("X-Real-Ip")
	if ipaddr == "" {
		ipaddr = req.Header.Get("X-Forwarded-For")
	}

	if ipaddr == "" {
		ipaddr = req.RemoteAddr
	}

	return ipaddr
}

func clientIPAddrAllowed(s string) bool {
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	ip := net.ParseIP(s[:strings.LastIndex(s, ":")])

	if ip.IsLoopback() {
		return true
	}

	var privateIPBlocks []*net.IPNet
	_, blockip4, _ := net.ParseCIDR("127.0.0.0/8")
	_, blockip6, _ := net.ParseCIDR("::1/128")
	privateIPBlocks = append(privateIPBlocks, blockip4, blockip6)

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

func (sin *app) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	http.ServeFile(w, r, "./index.html")
}

func (sin *app) console(w http.ResponseWriter, r *http.Request) {
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
	if clientIPAddr(r) != "10.10.10.1" {
		w.Header().Add("X-Authorize", "Users can only come from 10.10.10.0/24")
		w.WriteHeader(500)
		fmt.Fprintf(w, "500 internal server error")
	} else {
		fmt.Fprintf(w, RobotsTxt)
	}
}

var RobotsTxt = `
User-agent: *
Disallow: /h1dd3n,/admin
`

func (sin *app) h1dd3n(w http.ResponseWriter, r *http.Request) {
	ref := r.Referer()
	if ref != "p0st3b7n" {
		fmt.Fprintf(w, `Access disallowed. You are visiting from "" while authorized users should come only from a secret client!`)
	} else {
		fmt.Fprintf(w, fmt.Sprint(1))
	}
}
