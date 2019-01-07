package corehttp

import (
	"fmt"
	"net"
	"net/http"

	version "github.com/ipsn/go-ipfs"
	core "github.com/ipsn/go-ipfs/core"
	coreapi "github.com/ipsn/go-ipfs/core/coreapi"
	options "github.com/ipsn/go-ipfs/core/coreapi/interface/options"

	id "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type GatewayConfig struct {
	Headers      map[string][]string
	Writable     bool
	PathPrefixes []string
}

func GatewayOption(writable bool, paths ...string) ServeOption {
	return func(n *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		cfg, err := n.Repo.Config()
		if err != nil {
			return nil, err
		}

		api, err := coreapi.NewCoreAPI(n, options.Api.FetchBlocks(!cfg.Gateway.NoFetch))
		if err != nil {
			return nil, err
		}

		gateway := newGatewayHandler(n, GatewayConfig{
			Headers:      cfg.Gateway.HTTPHeaders,
			Writable:     writable,
			PathPrefixes: cfg.Gateway.PathPrefixes,
		}, api)

		for _, p := range paths {
			mux.Handle(p+"/", gateway)
		}
		return mux, nil
	}
}

func VersionOption() ServeOption {
	return func(_ *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Commit: %s\n", version.CurrentCommit)
			fmt.Fprintf(w, "Client Version: %s\n", id.ClientVersion)
			fmt.Fprintf(w, "Protocol Version: %s\n", id.LibP2PVersion)
		})
		return mux, nil
	}
}
