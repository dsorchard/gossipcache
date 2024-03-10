package gossipcache

import (
	"errors"
	"fmt"
	"github.com/hashicorp/memberlist"
	"net/http"

	"log"
	"os"
)

var _ memberlist.EventDelegate = &GossipCache{}

type GossipCache struct {
	GroupCachePool *HTTPPool
	Memberlist     *memberlist.Memberlist

	self   string
	peers  []string
	scheme string // http or https
	port   string // port number = none

	logger *log.Logger
}

func NewGossipHTTPPool() (*GossipCache, error) {
	var err error
	ac := GossipCache{}
	ac.logger = log.New(os.Stderr, "", log.LstdFlags)
	ac.scheme = "http"
	ac.port = ""

	// create memberlist
	mlConfig := memberlist.DefaultLANConfig()
	mlConfig.Events = &ac
	mlConfig.Logger = ac.logger
	if ac.Memberlist, err = memberlist.Create(mlConfig); err != nil {
		return nil, fmt.Errorf("gossipcache: can't create memberlist: %w", err)
	}

	// register self.
	if len(ac.Memberlist.Members()) == 0 {
		return nil, errors.New("memberlist can't find self")
	}
	self := ac.Memberlist.Members()[0]
	if self.Addr == nil {
		return nil, errors.New("self addr cannot be nil")
	}
	ac.self = self.Addr.String()
	ac.logger.Printf("gossipcache: self addr is: %s", ac.self)

	// create groupcache pool
	gcSelf := ac.groupCacheURL(ac.self)
	ac.GroupCachePool = newHTTPPoolOpts(gcSelf)
	return &ac, nil
}

func (ac *GossipCache) NotifyJoin(node *memberlist.Node) {
	uri := ac.groupCacheURL(node.Addr.String())
	ac.removePeer(uri)
	ac.peers = append(ac.peers, uri)
	if ac.GroupCachePool != nil {
		ac.GroupCachePool.set(ac.peers...)
	}
}

func (ac *GossipCache) NotifyLeave(node *memberlist.Node) {
	uri := ac.groupCacheURL(node.Addr.String())
	ac.removePeer(uri)
	ac.GroupCachePool.set(ac.peers...)
}

func (ac *GossipCache) NotifyUpdate(node *memberlist.Node) {
	ac.logger.Printf("GossipCache/NotifyUpdate: %+v", node)
}

func (ac *GossipCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ac.GroupCachePool.ServeHTTP(w, r)
}

func (ac *GossipCache) JoinOtherGossipNodes(existing []string) (int, error) {
	if ac.Memberlist == nil {
		return 0, errors.New("memberlist cannot be nil")
	}
	existing = append(existing, ac.self)
	return ac.Memberlist.Join(existing)
}

func (ac *GossipCache) StartHttpServer() {
	http.Handle(ac.GroupCachePool.opts.BasePath, ac)
}

//--------------------------------utils-------------------------------------

func (ac *GossipCache) groupCacheURL(addr string) string {
	u := fmt.Sprintf("%s://%s", ac.scheme, addr)
	return u
}

func (ac *GossipCache) removePeer(uri string) {
	var newPeers []string
	for _, peer := range ac.peers {
		if peer != uri {
			newPeers = append(newPeers, peer)
		}
	}
	ac.peers = newPeers
}
