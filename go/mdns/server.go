package mdns

import (
	"fmt"

	dbus "github.com/godbus/dbus/v5"
	avahi "github.com/holoplot/go-avahi"
)

// Server is a wrapper to the system's mDNS implementation
type Server struct {
	dbusConn     *dbus.Conn
	avahiServer  *avahi.Server
	hostname     string
	hostnameFqdn string
}

// ServerNew returns a new Server for mDNS operations
func ServerNew() (*Server, error) {
	s := new(Server)

	var err error

	s.dbusConn, err = dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	s.avahiServer, err = avahi.ServerNew(s.dbusConn)
	if err != nil {
		return nil, fmt.Errorf("avahi.ServerNew() failed: %v", err)
	}

	s.hostname, err = s.avahiServer.GetHostName()
	if err != nil {
		return nil, fmt.Errorf("GetHostName() failed: %v", err)
	}

	s.hostnameFqdn, err = s.avahiServer.GetHostNameFqdn()
	if err != nil {
		return nil, fmt.Errorf("GetHostNameFqdn() failed: %v", err)
	}

	return s, nil
}

// Close ...
func (s *Server) Close() {
	s.avahiServer.Close()
}

// Hostname returns the short (non-FQDN) hostname of this server
func (s *Server) Hostname() string {
	return s.hostname
}

// PublishRecord publishes info about our services
func (s *Server) PublishRecord(serviceName, serviceType, serviceSubtype string, port uint16, txt [][]byte) (*PublishedService, error) {
	eg, err := s.avahiServer.EntryGroupNew()
	if err != nil {
		return nil, fmt.Errorf("EntryGroupNew() failed: %v", err)
	}

	err = eg.AddService(avahi.InterfaceUnspec, avahi.ProtoUnspec, 0, serviceName, serviceType, "local", s.hostnameFqdn, port, txt)
	if err != nil {
		return nil, fmt.Errorf("AddService() failed: %v", err)
	}

	if len(serviceSubtype) > 0 {
		err := eg.AddServiceSubtype(avahi.InterfaceUnspec, avahi.ProtoUnspec, 0, serviceName, serviceType, "local", serviceSubtype)
		if err != nil {
			return nil, err
		}
	}

	err = eg.Commit()
	if err != nil {
		return nil, fmt.Errorf("Commit() failed: %v", err)
	}

	return &PublishedService{
		as: s.avahiServer,
		eg: eg,
	}, nil
}

// PublishedService is returned by PublishRecord()
type PublishedService struct {
	as *avahi.Server
	eg *avahi.EntryGroup
}

// Close tears down the published service
func (ps *PublishedService) Close() {
	ps.as.EntryGroupFree(ps.eg)
}
