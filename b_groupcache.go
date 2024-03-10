package gossipcache

import (
	"context"
	"errors"
	pb "github.com/golang/groupcache/groupcachepb"
	"math/rand"
	"sync"
)

var (
	// This global variable is the CORE connecting httpServer and GroupCache.
	// TODO: I think, it is a bad design. But what do i know?
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

//----------------------------------------------------------

type Group struct {
	peersOnce sync.Once
	peers     PeerPicker

	mainCache cache
	hotCache  cache

	name       string
	getter     Getter
	cacheBytes int64 // limit for sum of mainCache and hotCache size
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	return newGroup(name, cacheBytes, getter, nil)
}

// If peers is nil, the peerPicker is called via a sync.Once to initialize it.
func newGroup(name string, cacheBytes int64, getter Getter, peers PeerPicker) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, dup := groups[name]; dup {
		panic("duplicate registration of group " + name)
	}
	g := &Group{
		name:       name,
		getter:     getter,
		peers:      peers,
		cacheBytes: cacheBytes,
	}
	groups[name] = g
	return g
}

func (g *Group) initPeers() {
	if g.peers == nil {
		g.peers = getPeers(g.name)
	}
}

func (g *Group) lookupCache(key string) (value ByteView, ok bool) {
	if g.cacheBytes <= 0 {
		return
	}
	value, ok = g.mainCache.get(key)
	if ok {
		return
	}
	value, ok = g.hotCache.get(key)
	return
}

func (g *Group) Get(ctx context.Context, key string, dest Sink) error {
	g.peersOnce.Do(g.initPeers)
	if dest == nil {
		return errors.New("groupcache: nil dest Sink")
	}
	value, cacheHit := g.lookupCache(key)

	if cacheHit {
		return setSinkView(dest, value)
	}
	destPopulated := false
	value, destPopulated, err := g.load(ctx, key, dest)
	if err != nil {
		return err
	}
	if destPopulated {
		return nil
	}
	return setSinkView(dest, value)
}

// load loads key either by invoking the getter locally or by sending it to another machine.
func (g *Group) load(ctx context.Context, keyVal string, dest Sink) (value ByteView, destPopulated bool, err error) {
	viewi, err := func(key string) (interface{}, error) {
		if value, cacheHit := g.lookupCache(key); cacheHit {
			return value, nil
		}
		var value ByteView
		var err error
		if peer, ok := g.peers.PickPeer(key); ok {
			value, err = g.getFromPeer(ctx, peer, key)
			if err == nil {
				return value, nil
			}
		}
		value, err = g.getLocally(ctx, key, dest)
		if err != nil {
			return nil, err
		}
		destPopulated = true // only one caller of load gets this return value
		g.populateCache(key, value, &g.mainCache)
		return value, nil
	}(keyVal)

	if err == nil {
		value = viewi.(ByteView)
	}
	return
}

func (g *Group) getFromPeer(ctx context.Context, peer ProtoGetter, key string) (ByteView, error) {
	req := &pb.GetRequest{
		Group: &g.name,
		Key:   &key,
	}
	res := &pb.GetResponse{}
	err := peer.Get(ctx, req, res)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: res.Value}
	if rand.Intn(10) == 0 {
		g.populateCache(key, value, &g.hotCache)
	}
	return value, nil
}

func (g *Group) getLocally(ctx context.Context, key string, dest Sink) (ByteView, error) {
	err := g.getter.Get(ctx, key, dest)
	if err != nil {
		return ByteView{}, err
	}
	return dest.view()
}

func (g *Group) populateCache(key string, value ByteView, cache *cache) {
	if g.cacheBytes <= 0 {
		return
	}
	cache.add(key, value)
	for {
		mainBytes := g.mainCache.bytes()
		hotBytes := g.hotCache.bytes()
		if mainBytes+hotBytes <= g.cacheBytes {
			return
		}

		victim := &g.mainCache
		if hotBytes > mainBytes/8 {
			victim = &g.hotCache
		}
		victim.removeOldest()
	}
}
