package gossipcache

import (
	"github.com/charmbracelet/log"
	"github.com/golang/groupcache/consistenthash"
	pb "github.com/golang/groupcache/groupcachepb"
	"github.com/golang/protobuf/proto"
	"net/http"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_groupcache/"
	defaultReplicas = 50
)

var httpPoolLogger = log.WithPrefix("httpPool")

type HTTPPool struct {
	self string
	opts struct {
		BasePath string
		Replicas int
		HashFn   consistenthash.Hash
	}

	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

func newHTTPPool(self string) *HTTPPool {
	p := newHTTPPoolOpts(self)
	http.Handle(p.opts.BasePath, p)
	return p
}

func newHTTPPoolOpts(self string) *HTTPPool {
	p := &HTTPPool{
		self:        self,
		httpGetters: make(map[string]*httpGetter),
	}
	p.opts.BasePath = defaultBasePath
	p.opts.Replicas = defaultReplicas
	p.peers = consistenthash.New(p.opts.Replicas, p.opts.HashFn)
	RegisterPeerPicker(func() PeerPicker { return p })
	return p
}

func (p *HTTPPool) set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(p.opts.Replicas, p.opts.HashFn)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.opts.BasePath}
	}
}

func (p *HTTPPool) PickPeer(key string) (peer ProtoGetter, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.peers.IsEmpty() {
		return nil, false
	}
	if peerKey := p.peers.Get(key); peerKey != p.self {
		httpPoolLogger.Infof("HTTPPool.PickPeer: peerKey=%s, self=%s", peerKey, p.self)
		return p.httpGetters[peerKey], true
	}
	return nil, false
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse request.
	if !strings.HasPrefix(r.URL.Path, p.opts.BasePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	parts := strings.SplitN(r.URL.Path[len(p.opts.BasePath):], "/", 2)
	groupName := parts[0]
	key := parts[1]

	// Get the value for this key.
	group := GetGroup(groupName)
	ctx := r.Context()
	var value []byte
	err := group.Get(ctx, key, AllocatingByteSliceSink(&value))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpPoolLogger.Infof("HTTPPool.ServeHTTP: key=%s, value=%s", key, value)

	body, err := proto.Marshal(&pb.GetResponse{Value: value})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(body)
}

// ---------------------------utils---------------------------------
