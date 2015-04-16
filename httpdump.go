package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
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
	maxBytes                 = 102400
	maxLines                 = 100
	loopback                 = "127.0.0.1"
)

func defaultHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			if r.Method == "OPTIONS" {
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	listen := flag.String("listen", "127.0.0.1:8090", "The host and port to listen on.")
	flag.Parse()
	http.HandleFunc("/headers", headers)
	http.HandleFunc("/status/", status)
	http.HandleFunc("/ip", ip)
	http.HandleFunc("/get", get)
	http.Handle("/gzip", gzipped.New(http.HandlerFunc(gzip)))
	http.HandleFunc("/user-agent", userAgent)
	http.HandleFunc("/bytes/", writeBytes)
	http.HandleFunc("/stream/", stream)
	log.Fatal(http.ListenAndServe(*listen, defaultHandler(http.DefaultServeMux)))
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

func getOrigin(r *http.Request) string {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" && forwarded != host {
		if host == loopback {
			return forwarded
		}
		host = fmt.Sprintf("%s, %s", forwarded, host)
	}
	return host
}

func ip(w http.ResponseWriter, r *http.Request) {
	var o struct {
		Origin string `json:"origin"`
	}
	o.Origin = getOrigin(r)
	writeJSON(w, o, http.StatusOK)
}

type request struct {
	Args    url.Values  `json:"args"`
	Gzipped bool        `json:"gzipped,omitempty"`
	Headers http.Header `json:"headers"`
	Origin  string      `json:"origin"`
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
	writeJSON(w, getReq(r), http.StatusOK)
}

func gzip(w http.ResponseWriter, r *http.Request) {
	req := getReq(r)
	if _, ok := w.(gzipped.GzipResponseWriter); ok {
		req.Gzipped = true
	}
	writeJSON(w, req, http.StatusOK)
}

func userAgent(w http.ResponseWriter, r *http.Request) {
	var resp struct {
		UserAgent string `json:"user-agent"`
	}
	resp.UserAgent = r.Header.Get("User-Agent")
	writeJSON(w, resp, http.StatusOK)
}

func writeBytes(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil || n < 0 || n > maxBytes {
		http.Error(w, fmt.Sprintf("number of bytes must be in range: 0 - %d", maxBytes), http.StatusBadRequest)
		return
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func stream(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil || n < 0 {
		http.Error(w, errWantInteger, http.StatusBadRequest)
		return
	}
	n = min(n, maxLines)
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
