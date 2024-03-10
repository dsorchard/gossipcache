package gossipcache

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/hashicorp/memberlist"
	"net/http"
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
}

func NewGossipHTTPPool(gossipPort int, httpPort int) (*GossipCache, error) {
	var err error
	ac := GossipCache{}
	ac.httpPort = httpPort
	ac.host = "127.0.0.1"

	// 1. create memberlist
	mlConfig := memberlist.DefaultLocalConfig()
	mlConfig.Events = &ac
	mlConfig.BindAddr = "127.0.0.1"
	mlConfig.BindPort = gossipPort
	mlConfig.Name = fmt.Sprintf("%d", httpPort)
	mlConfig.LogOutput = NewMemberlistLogger()
	if ac.Memberlist, err = memberlist.Create(mlConfig); err != nil {
		return nil, fmt.Errorf("gossipcache: can't create memberlist: %w", err)
	}

	// create groupcache pool
	httpUrl := ac.httpGroupCacheURL(fmt.Sprintf("%d", ac.httpPort))
	ac.GroupCachePool = newHTTPPool(httpUrl)
	return &ac, nil
}

func (ac *GossipCache) NotifyJoin(node *memberlist.Node) {
	httpUrl := ac.httpGroupCacheURL(node.Name)
	ac.removePeer(httpUrl)
	ac.peers = append(ac.peers, httpUrl)
	log.Printf("GossipCache/NotifyJoin: %+v", ac.peers)
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
	return fmt.Sprintf("http://%s:%s", ac.host, port)
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

// --------------------------------utils-------------------------------------

type MemberlistLogger struct {
	Logger *log.Logger
}

func NewMemberlistLogger() MemberlistLogger {
	return MemberlistLogger{
		Logger: log.NewWithOptions(os.Stderr, log.Options{
			Prefix: "memberlist",
		}),
	}
}

func (l MemberlistLogger) Write(p []byte) (n int, err error) {
	l.Logger.Debug(string(p)) // change it to `Info` to see the memberlist logs
	return len(p), nil
}
