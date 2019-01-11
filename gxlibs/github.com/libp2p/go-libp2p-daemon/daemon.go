package p2pd

import (
	"context"
	"fmt"
	"sync"

	logging "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-log"
	libp2p "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p"
	host "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-host"
	dht "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-kad-dht/opts"
	peer "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-peer"
	proto "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-protocol"
	ps "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-pubsub"
	rhost "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/ipsn/go-ipfs/gxlibs/github.com/multiformats/go-multiaddr"
	manet "github.com/ipsn/go-ipfs/gxlibs/github.com/multiformats/go-multiaddr-net"
)

var log = logging.Logger("p2pd")

type Daemon struct {
	ctx      context.Context
	host     host.Host
	listener manet.Listener

	dht    *dht.IpfsDHT
	pubsub *ps.PubSub

	mx sync.Mutex
	// stream handlers: map of protocol.ID to multi-address
	handlers map[proto.ID]ma.Multiaddr
}

func NewDaemon(ctx context.Context, maddr ma.Multiaddr, opts ...libp2p.Option) (*Daemon, error) {
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	l, err := manet.Listen(maddr)
	if err != nil {
		h.Close()
		return nil, err
	}

	d := &Daemon{
		ctx:      ctx,
		host:     h,
		listener: l,
		handlers: make(map[proto.ID]ma.Multiaddr),
	}

	go d.listen()

	return d, nil
}

func (d *Daemon) EnableDHT(client bool) error {
	var opts []dhtopts.Option

	if client {
		opts = append(opts, dhtopts.Client(true))
	}

	dht, err := dht.New(d.ctx, d.host, opts...)
	if err != nil {
		return err
	}

	d.dht = dht
	d.host = rhost.Wrap(d.host, d.dht)

	return nil
}

func (d *Daemon) EnablePubsub(router string, sign, strict bool) error {
	var opts []ps.Option

	if sign {
		opts = append(opts, ps.WithMessageSigning(sign))

		if strict {
			opts = append(opts, ps.WithStrictSignatureVerification(strict))
		}
	}

	switch router {
	case "floodsub":
		pubsub, err := ps.NewFloodSub(d.ctx, d.host, opts...)
		if err != nil {
			return err
		}
		d.pubsub = pubsub
		return nil

	case "gossipsub":
		pubsub, err := ps.NewGossipSub(d.ctx, d.host, opts...)
		if err != nil {
			return err
		}
		d.pubsub = pubsub
		return nil

	default:
		return fmt.Errorf("unknown pubsub router: %s", router)
	}

}

func (d *Daemon) ID() peer.ID {
	return d.host.ID()
}

func (d *Daemon) Addrs() []ma.Multiaddr {
	return d.host.Addrs()
}

func (d *Daemon) listen() {
	for {
		c, err := d.listener.Accept()
		if err != nil {
			log.Errorf("error accepting connection: %s", err.Error())
		}

		log.Debug("incoming connection")
		go d.handleConn(c)
	}
}
