package discovery

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"

	"github.com/bloxapp/ssv/logging"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	mdnsDiscover "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	// LocalDiscoveryServiceTag is used in our mDNS advertisements to discover other peers
	LocalDiscoveryServiceTag = "ssv.discovery"
)

// localDiscovery implements ssv_discovery.Service using mDNS and KAD-DHT
type localDiscovery struct {
	ctx        context.Context
	svc        mdnsDiscover.Service
	disc       discovery.Discovery
	routingTbl routing.Routing

	host host.Host
}

// NewLocalDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func NewLocalDiscovery(ctx context.Context, logger *zap.Logger, host host.Host) (Service, error) {
	logger = logger.Named(logging.NameDiscoveryService)
	logger.Debug("configuring mdns")

	routingDHT, disc, err := NewKadDHT(ctx, host, dht.ModeServer)
	if err != nil {
		return nil, errors.Wrap(err, "could not create DHT")
	}

	return &localDiscovery{
		ctx:        ctx,
		host:       host,
		routingTbl: routingDHT,
		disc:       disc,
		svc: mdnsDiscover.NewMdnsService(host, LocalDiscoveryServiceTag, &discoveryNotifee{
			handler: handle(host, func(e PeerEvent) {
				err := host.Connect(ctx, e.AddrInfo)
				if err != nil {
					logger.Warn("could not connect to peer", zap.Any("addrInfo", e.AddrInfo), zap.Error(err))
					return
				}
				logger.Debug("connected new peer", zap.Any("addrInfo", e.AddrInfo))
			}),
		}),
	}, nil
}

func handle(host host.Host, handler HandleNewPeer) HandleNewPeer {
	return func(e PeerEvent) {
		ctns := host.Network().Connectedness(e.AddrInfo.ID)
		switch ctns {
		case libp2pnetwork.CannotConnect, libp2pnetwork.Connected:
		default:
			go handler(e)
		}
	}
}

// Bootstrap starts to listen to new nodes
func (md *localDiscovery) Bootstrap(logger *zap.Logger, handler HandleNewPeer) error {
	err := md.svc.Start()
	if err != nil {
		return errors.Wrap(err, "could not start mdns service")
	}
	return md.routingTbl.Bootstrap(md.ctx)
}

// Advertise implements discovery.Advertiser
func (md *localDiscovery) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	return md.disc.Advertise(ctx, ns, opt...)
}

// FindPeers implements discovery.Discoverer
func (md *localDiscovery) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	return md.disc.FindPeers(ctx, ns, opt...)
}

// RegisterSubnets implements Service
func (md *localDiscovery) RegisterSubnets(logger *zap.Logger, subnets ...int) error {
	// TODO
	return nil
}

// DeregisterSubnets implements Service
func (md *localDiscovery) DeregisterSubnets(logger *zap.Logger, subnets ...int) error {
	// TODO
	return nil
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	handler HandleNewPeer
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.handler(PeerEvent{AddrInfo: pi})
}

func (md *localDiscovery) Close() error {
	if err := md.svc.Close(); err != nil {
		return err
	}
	return nil
}

// Mock method to follow interface.
func (md *localDiscovery) Node(logger *zap.Logger, info peer.AddrInfo) (*enode.Node, error) {
	return nil, nil
}
