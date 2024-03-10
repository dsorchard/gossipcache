### Run

```shell
go build
```

```shell
./simple -gossip=8000 -http=8001 -seed=http://127.0.0.1:8000
```

```shell
./simple -gossip=9000 -http=9001 -seed=http://127.0.0.1:8000
```

```shell
./simple -gossip=10000 -http=10001 -seed=http://127.0.0.1:8000
```

```shell
curl -Ss -XGET "localhost:8001/color?name=black"
```