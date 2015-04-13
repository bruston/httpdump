package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/bruston/handlers/gzipped"
)

const (
	errWantInteger           = "n must be an integer"
	errStreamingNotSupported = "your client does not support streaming"
)

func main() {
	listen := flag.String("listen", ":8090", "The host and port to listen on.")
	flag.Parse()
	http.HandleFunc("/headers", headers)
	http.HandleFunc("/status/", status)
	http.HandleFunc("/ip", ip)
	http.HandleFunc("/get", get)
	http.Handle("/gzip", gzipped.New(http.HandlerFunc(gzippedResponse)))
	http.HandleFunc("/user-agent", useragent)
	http.HandleFunc("/bytes/", writeBytes)
	http.HandleFunc("/stream/", stream)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func jsonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-type", "application/json")
}

func writeJSON(w http.ResponseWriter, data interface{}, code int) error {
	jsonHeader(w)
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
	Gzipped bool        `json:"gzipped,omitempty"`
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

func getReq(r *http.Request) request {
	return request{
		Args:    r.URL.Query(),
		Headers: r.Header,
		Origin:  getOrigin(r),
		URL:     rawURL(r),
	}
}

func get(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	req := getReq(r)
	writeJSON(w, req, http.StatusOK)
}

func gzippedResponse(w http.ResponseWriter, r *http.Request) {
	req := getReq(r)
	if _, ok := w.(gzipped.GzipResponseWriter); ok {
		req.Gzipped = true
	}
	writeJSON(w, req, http.StatusOK)
}

func useragent(w http.ResponseWriter, r *http.Request) {
	var resp struct {
		UserAgent string `json:"user-agent"`
	}
	resp.UserAgent = r.Header.Get("User-Agent")
	writeJSON(w, resp, http.StatusOK)
}

func writeBytes(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		http.Error(w, errWantInteger, http.StatusBadRequest)
		return
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func stream(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		http.Error(w, errWantInteger, http.StatusBadRequest)
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, errStreamingNotSupported, http.StatusBadRequest)
		return
	}
	req := getReq(r)
	jsonHeader(w)
	for i := 0; i < n; i++ {
		if err := json.NewEncoder(w).Encode(req); err != nil {
			return
		}
		f.Flush()
	}
}
