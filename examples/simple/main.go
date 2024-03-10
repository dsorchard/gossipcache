package main

import (
	"context"
	"errors"
	"flag"
	"gossipcache"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var Store = map[string][]byte{
	"red":   []byte("#FF0000"),
	"green": []byte("#00FF00"),
	"blue":  []byte("#0000FF"),
}

func main() {
	iport, _ := strconv.Atoi(*flag.String("iport", "8000", "gossip address"))
	eport := *flag.String("eport", ":8001", "http list")
	seed := strings.Split(*flag.String("seed", "127.0.0.1:8000", "server pool list"), ",")
	flag.Parse()

	// Create group first. this group is going to be registered in the global map variable.
	gc := gossipcache.NewGroup("foo", 64<<20, gossipcache.GetterFunc(
		func(ctx context.Context, key string, dest gossipcache.Sink) error {
			log.Println("looking up", key)
			v, ok := Store[key]
			if !ok {
				return errors.New("color not found")
			}
			_ = dest.SetBytes(v)
			return nil
		}))

	// then run the http server.
	pool, err := gossipcache.NewGossipHTTPPool(iport)
	if err != nil {
		log.Fatal(err)
	}
	_, err = pool.JoinGossipCluster(seed)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/color", func(w http.ResponseWriter, r *http.Request) {
		color := r.FormValue("name")
		var b []byte
		err = gc.Get(nil, color, gossipcache.AllocatingByteSliceSink(&b))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
		_, _ = w.Write([]byte{'\n'})
	})
	_ = http.ListenAndServe(eport, nil)
}
