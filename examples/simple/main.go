package main

import (
	"context"
	"flag"
	"gossipcache"
	"log"
	"net/http"
	"strings"
)

const (
	defaultAddr = ":http"
)

var Store = map[string][]byte{
	"red":   []byte("#FF0000"),
	"green": []byte("#00FF00"),
	"blue":  []byte("#0000FF"),
}

var gc *gossipcache.Group

func main() {
	existing := strings.Split(*flag.String("pool", "http://localhost:8080", "server pool list"), ",")
	flag.Parse()

	// Create group first.
	gc = gossipcache.NewGroup("foo", 64<<20, gossipcache.GetterFunc(
		func(ctx context.Context, key string, dest gossipcache.Sink) error {
			log.Printf("Fetching: %s", key)
			_ = dest.SetString(key)
			return nil
		}))

	// then run the http server, so that the group is registered in the global map variable.
	// and now it is accessible from the http handler.
	ac, _ := gossipcache.NewGossipHTTPPool()
	_, _ = ac.JoinOtherGossipNodes(existing)
	ac.StartHttpServer()

	http.HandleFunc("/color", func(w http.ResponseWriter, r *http.Request) {
		color := r.FormValue("name")
		var b []byte
		err := gc.Get(nil, color, gossipcache.AllocatingByteSliceSink(&b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Write(b)
		w.Write([]byte{'\n'})
	})
}
