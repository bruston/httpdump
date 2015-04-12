package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

func main() {
	listen := flag.String("listen", ":8090", "The host and port to listen on.")
	flag.Parse()

	http.HandleFunc("/headers", headers)
	http.HandleFunc("/status/", status)
	http.HandleFunc("/ip", ip)
	http.HandleFunc("/get", get)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func writeJSON(w http.ResponseWriter, data interface{}, code int) error {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(data)
}

func headers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r.Header, http.StatusOK)
}

func status(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		http.Error(w, "status code must be an integer", http.StatusBadRequest)
		return
	}
	w.WriteHeader(code)
}

type origin struct {
	IP           string `json:"ip"`
	ForwardedFor string `json:"forwarded_for,omitempty"`
}

func getOrigin(r *http.Request) origin {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return origin{host, r.Header.Get("X-Forwarded-For")}
}

func ip(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, getOrigin(r), http.StatusOK)
}

type request struct {
	Args    url.Values  `json:"args"`
	Headers http.Header `json:"headers"`
	Origin  origin      `json:"origin"`
	URL     string      `json:"url"`
}

func rawURL(r *http.Request) string {
	var scheme string
	if r.TLS == nil {
		scheme = "http"
	} else {
		scheme = "https"
	}
	return scheme + "://" + r.Host + r.URL.String()
}

func get(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	req := request{
		Args:    r.URL.Query(),
		Headers: r.Header,
		Origin:  getOrigin(r),
		URL:     rawURL(r),
	}
	writeJSON(w, req, http.StatusOK)
}
