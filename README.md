# Group Cache + Gossip Protocol

This is a simple modification of group cache to support gossip protocol.

### Reference

- [Group Cache](https://github.com/golang/groupcache)
- [AutoCache](https://github.com/pomerium/autocache)
- [Group Cache Example](https://gist.github.com/fiorix/816117cfc7573319b72d)

### NOTES

- Key <-> Vnode mapping strategies.
    - Here we partition keys based on consistent hashing. so key <-> vnode mapping is automatic.
    - We could also keep a mapping of key <-> vnode and use that to find the node for a given key. For this you need
      propagate the mapping info in gossip protocol. This is implemented
      in [MatrixOrigin](https://github.com/arjunsk/matrixone/blob/970b3b8bca5b91f5f8c98618b80ac941a21b286f/pkg/gossip/node.go#L177).
- We use a global map variable to connect HttpPool and GroupCache (map[groupName]*Group).
- I don't think we do any kind of redistribution of data since it is a cache.
