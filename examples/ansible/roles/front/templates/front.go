package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var port = flag.Int("port", 8080, "listen port")
var apphost = flag.String("app", "", "app host")

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(*apphost)
	resp, err := http.Get(*apphost)
	if err != nil {
		fmt.Fprintf(w, "Error!!! %s\n", err.Error())
	}
	fmt.Println(resp)
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(w, "front: app response = %s\n", string(body))
}

func newLogHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(fmt.Sprintf(":%d",*port), newLogHandler(http.DefaultServeMux))
	if err != nil {
		println(err.Error)
	}
}