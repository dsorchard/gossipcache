package gossipcache

// Peer Picker Strategies: ConsistentHashing or Keeping Key-Server mapping.

type PeerPicker interface {
	PickPeer(key string) (peer ProtoGetter, ok bool)
}

//-------------------------------------HTTPPool-------------------------------------

var (
	portPicker func(groupName string) PeerPicker
)

func RegisterPeerPicker(fn func() PeerPicker) {
	if portPicker != nil {
		panic("RegisterPeerPicker called more than once")
	}
	portPicker = func(_ string) PeerPicker { return fn() }
}

func getPeers(groupName string) PeerPicker {
	if portPicker == nil {
		return NoPeers{}
	}
	pk := portPicker(groupName)
	if pk == nil {
		pk = NoPeers{}
	}
	return pk
}

//-------------------------------------NoPeers-------------------------------------

type NoPeers struct{}

func (NoPeers) PickPeer(key string) (peer ProtoGetter, ok bool) { return }
