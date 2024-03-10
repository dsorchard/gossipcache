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

	host              string
	httpPort          int
	hostGossipAddress string
	peers             []string

	logger *log.Logger
}

func NewGossipHTTPPool(gossipPort int, httpPort int) (*GossipCache, error) {
	var err error
	ac := GossipCache{}
	ac.logger = log.New(os.Stderr, "", log.LstdFlags)
	ac.httpPort = httpPort
	ac.host = "127.0.0.1"

	// 1. create memberlist
	mlConfig := memberlist.DefaultLocalConfig()
	mlConfig.Events = &ac
	mlConfig.BindAddr = "127.0.0.1"
	mlConfig.BindPort = gossipPort
	mlConfig.Name = fmt.Sprintf("%d", mlConfig.BindPort)
	mlConfig.Logger = ac.logger
	if ac.Memberlist, err = memberlist.Create(mlConfig); err != nil {
		return nil, fmt.Errorf("gossipcache: can't create memberlist: %w", err)
	}

	// create groupcache pool
	httpUrl := ac.httpGroupCacheURL(fmt.Sprintf("%d", ac.httpPort))
	ac.GroupCachePool = newHTTPPool(httpUrl)
	return &ac, nil
}

func (ac *GossipCache) NotifyJoin(node *memberlist.Node) {
	uri := ac.httpGroupCacheURL(node.Name)
	ac.removePeer(uri)
	ac.peers = append(ac.peers, uri)
	if ac.GroupCachePool != nil {
		ac.GroupCachePool.set(ac.peers...)
	}
}

func (ac *GossipCache) NotifyLeave(node *memberlist.Node) {
	uri := ac.httpGroupCacheURL(node.Name)
	ac.removePeer(uri)
	ac.GroupCachePool.set(ac.peers...)
}

func (ac *GossipCache) NotifyUpdate(node *memberlist.Node) {
	ac.logger.Printf("GossipCache/NotifyUpdate: %+v", node)
}

func (ac *GossipCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ac.GroupCachePool.ServeHTTP(w, r)
}

func (ac *GossipCache) JoinGossipCluster(existing []string) (int, error) {
	if ac.Memberlist == nil {
		return 0, errors.New("memberlist cannot be nil")
	}
	existing = append(existing, ac.hostGossipAddress)
	return ac.Memberlist.Join(existing)
}

//--------------------------------utils-------------------------------------

func (ac *GossipCache) httpGroupCacheURL(port string) string {
	return fmt.Sprintf("%s:%s", ac.host, port)
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
