package servers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type ServerInfo struct {
	HTTP            *http.Server
	HTTPS           *http.Server
	HTTP_v6         *http.Server
	HTTPS_v6        *http.Server
	HTTPSKeyFile    string
	HTTPSCertFile   string
	HasBothHandlers bool
	FastShutdown    bool
}

func New(httpPort, httpsPort, HTTPSKeyFile, HTTPSCertFile string) *ServerInfo {
	servers := ServerInfo{}
	if httpsPort != "" && HTTPSKeyFile != "" && HTTPSCertFile != "" {
		// Run HTTPS listener when port, key and cert are specified
		// This is default in operator deployments
		servers.HTTPS = &http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%s", httpsPort),
			ReadHeaderTimeout: 3 * time.Second,
		}
		servers.HTTPS_v6 = &http.Server{
			Addr:              fmt.Sprintf("[::]:%s", httpsPort),
			ReadHeaderTimeout: 3 * time.Second,
		}
		servers.HTTPSCertFile = HTTPSCertFile
		servers.HTTPSKeyFile = HTTPSKeyFile
	} else if httpPort == "" {
		// Run HTTP listener on HTTPS port if httpPort is not set
		// This is default in podman deployment
		servers.HTTP = &http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%s", httpsPort),
			ReadHeaderTimeout: 3 * time.Second,
		}
		servers.HTTP_v6 = &http.Server{
			Addr:              fmt.Sprintf("[::]:%s", httpsPort),
			ReadHeaderTimeout: 3 * time.Second,
		}
	}
	if httpPort != "" {
		// Run HTTP listener if httpPort is set
		servers.HTTP = &http.Server{
			Addr:              fmt.Sprintf("0.0.0.0:%s", httpPort),
			ReadHeaderTimeout: 3 * time.Second,
		}
		servers.HTTP_v6 = &http.Server{
			Addr:              fmt.Sprintf("[::]:%s", httpPort),
			ReadHeaderTimeout: 3 * time.Second,
		}
	}
	servers.HasBothHandlers = servers.HTTP != nil && servers.HTTPS != nil
	return &servers
}

func shutdown(name string, server *http.Server) {
	if err := server.Shutdown(context.TODO()); err != nil {
		log.Infof("%s shutdown failed: %v", name, err)
		if err := server.Close(); err != nil {
			log.Fatalf("%s emergency shutdown failed: %v", name, err)
		}
	} else {
		log.Infof("%s server terminated gracefully", name)
	}
}

func (s *ServerInfo) ListenAndServe() {
	if s.HTTP != nil {
		go httpListen(s.HTTP)
		go httpListen(s.HTTP_v6)
	}

	if s.HTTPS != nil {
		go httpListen(s.HTTPS)
		go httpListen(s.HTTPS_v6)
	}
}

func (s *ServerInfo) Shutdown() bool {
	if s.HTTPS != nil {
		if s.FastShutdown {
			s.HTTPS.Close()
		} else {
			shutdown("HTTPS", s.HTTPS)
		}
	}
	if s.HTTP != nil {
		if s.FastShutdown {
			s.HTTP.Close()
		} else {
			shutdown("HTTP", s.HTTP)
		}
	}
	return true
}

func httpListen(server *http.Server) {
	log.Infof("Starting handler on %s...", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Listener closed: %v", err)
	}
}
