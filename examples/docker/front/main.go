package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	appHost := os.Getenv("egress_app1")
	resp, err := http.Get(appHost)
	if err != nil {
		fmt.Fprintf(w, "Error!!! %s\n", err.Error())
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(w, "front: app1 response = %s\n", string(body))
}

func newLogHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", newLogHandler(http.DefaultServeMux))
}
