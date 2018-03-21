package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	client       = http.Client{}
	traceHeaders = []string{
		"X-Ot-Span-Context",
		"X-Request-Id",
		"X-B3-TraceId",
		"X-B3-SpanId",
		"X-B3-ParentSpanId",
		"X-B3-Sampled",
		"X-B3-Flags",
	}
)

func propagate(in *http.Request, out *http.Request) {
	for _, key := range traceHeaders {
		value := in.Header.Get(key)
		if value != "" {
			out.Header.Add(key, value)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	appHost := os.Getenv("egress_app1")
	req, err := http.NewRequest("GET", appHost, nil)
	if err != nil {
		fmt.Fprintf(w, "Error!!! %s\n", err.Error())
	}

	// copy headers for tracing
	propagate(r, req)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(w, "Error!!! %s\n", err.Error())
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(w, "front: app1 response = %s\n", string(body))
}

func newLogHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s header=%+v", r.RemoteAddr, r.Method, r.URL, r.Header)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", newLogHandler(http.DefaultServeMux))
}
