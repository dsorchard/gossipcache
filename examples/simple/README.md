### Run

```shell
go build
```

```shell
./simple -gossip=8000 -http=8001 -seed=127.0.0.1:8000
```

```shell
./simple -gossip=9000 -http=9001 -seed=127.0.0.1:8000
```

```shell
./simple -gossip=10000 -http=10001 -seed=127.0.0.1:8000
```

```shell
kill -9 $(lsof -t -i:8001)
kill -9 $(lsof -t -i:9001)
kill -9 $(lsof -t -i:10001)

```

```shell
curl -Ss -XGET "localhost:9001/color?name=green"
```

```shell
curl -Ss -XGET "localhost:8001/_groupcache/foo/green"
```