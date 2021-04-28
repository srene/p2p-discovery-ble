package discovery

import (
	"context"
//	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	logging "github.com/ipfs/go-log"
//	ma "github.com/multiformats/go-multiaddr"
//	manet "github.com/multiformats/go-multiaddr/net"
//	"github.com/whyrusleeping/mdns"
)

func init() {
	// don't let mdns use logging...
//	mdns.DisableLogging = true
}

var log = logging.Logger("blediscovery")

var DiscoveryMap sync.Map

const ServiceTag = "_ble-discovery._udp"

type Service interface {
	io.Closer
	RegisterNotifee(Notifee)
	UnregisterNotifee(Notifee)
}

type Notifee interface {
	HandlePeerFound(peer.AddrInfo)
}

type bleDiscoveryService struct {
//	server  *mdns.Server
//	service *mdns.MDNSService
	host    host.Host
	tag     string
	driver   NativeDriver
	lk       sync.Mutex
	notifees []Notifee
	interval time.Duration
}

func getDialableListenAddrs(ph host.Host) ([]*net.TCPAddr, error) {
	var out []*net.TCPAddr
	/*addrs, err := ph.Network().InterfaceListenAddresses()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		na, err := manet.ToNetAddr(addr)
		if err != nil {
			continue
		}
		tcp, ok := na.(*net.TCPAddr)
		if ok {
			out = append(out, tcp)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("failed to find good external addr from peerhost")
	}*/
	return out, nil
}

func NewBleDiscoveryService(ctx context.Context, peerhost host.Host,driver   NativeDriver,interval time.Duration, serviceTag string) (Service, error) {

	var ipaddrs []net.IP
	//port := 4001

	addrs, err := getDialableListenAddrs(peerhost)
	if err != nil {
		log.Warning(err)
	} else {
		//port = addrs[0].Port
		for _, a := range addrs {
			ipaddrs = append(ipaddrs, a.IP)
		}
	}

	if driver == nil {
		log.Error("error: NewTransport: driver is nil")
		driver = &NoopNativeDriver{}
	}

	//myid := peerhost.ID().Pretty()

	//info := []string{myid}
	if serviceTag == "" {
		serviceTag = ServiceTag
	}
	/*service, err := mdns.NewMDNSService(myid, serviceTag, "", "", port, ipaddrs, info)
	if err != nil {
		return nil, err
	}

	// Create the mDNS server, defer shutdown
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, err
	}*/

	s := &bleDiscoveryService{
		//server:   server,
		//service:  service,
		driver: driver,
		host:     peerhost,
		interval: interval,
		tag:      serviceTag,
	}

	DiscoveryMap.Store(s.driver.ProtocolName(), s)

	//go s.pollForEntries(ctx)

	return s, nil
}

func (m *bleDiscoveryService) Close() error {
	//return m.server.Shutdown()
	return nil
}

/*func (m *bleDiscoveryService) pollForEntries(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		//execute mdns query right away at method call and then with every tick
		entriesCh := make(chan *mdns.ServiceEntry, 16)
		go func() {
			for entry := range entriesCh {
				m.handleEntry(entry)
			}
		}()

		log.Debug("starting mdns query")
		qp := &mdns.QueryParam{
			Domain:  "local",
			Entries: entriesCh,
			Service: m.tag,
			Timeout: time.Second * 5,
		}

		err := mdns.Query(qp)
		if err != nil {
			log.Warnw("mdns lookup error", "error", err)
		}
		close(entriesCh)
		log.Debug("mdns query complete")

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Debug("mdns service halting")
			return
		}
	}
}*/

/*func (m *bleDiscoveryService) handleEntry(e *mdns.ServiceEntry) {
	log.Debugf("Handling MDNS entry: [IPv4 %s][IPv6 %s]:%d %s", e.AddrV4, e.AddrV6, e.Port, e.Info)
	mpeer, err := peer.IDB58Decode(e.Info)
	if err != nil {
		log.Warning("Error parsing peer ID from mdns entry: ", err)
		return
	}

	if mpeer == m.host.ID() {
		log.Debug("got our own mdns entry, skipping")
		return
	}

	var addr net.IP
	if e.AddrV4 != nil {
		addr = e.AddrV4
	} else if e.AddrV6 != nil {
		addr = e.AddrV6
	} else {
		log.Warning("Error parsing multiaddr from mdns entry: no IP address found")
		return
	}

	maddr, err := manet.FromNetAddr(&net.TCPAddr{
		IP:   addr,
		Port: e.Port,
	})
	if err != nil {
		log.Warning("Error parsing multiaddr from mdns entry: ", err)
		return
	}

	pi := peer.AddrInfo{
		ID:    mpeer,
		Addrs: []ma.Multiaddr{maddr},
	}

	m.lk.Lock()
	for _, n := range m.notifees {
		go n.HandlePeerFound(pi)
	}
	m.lk.Unlock()
}*/

func (m *bleDiscoveryService) RegisterNotifee(n Notifee) {
	m.lk.Lock()
	m.notifees = append(m.notifees, n)
	m.lk.Unlock()
}

func (m *bleDiscoveryService) UnregisterNotifee(n Notifee) {
	m.lk.Lock()
	found := -1
	for i, notif := range m.notifees {
		if notif == n {
			found = i
			break
		}
	}
	if found != -1 {
		m.notifees = append(m.notifees[:found], m.notifees[found+1:]...)
	}
	m.lk.Unlock()
}

// Protocols returns the set of protocols handled by this transport.
func (t *bleDiscoveryService) Protocols() []int {
	return []int{t.driver.ProtocolCode()}
}

func (t *bleDiscoveryService) String() string {
	return t.driver.ProtocolName()
}