package server

import (
	"strconv"

	"crypto/tls"
	"crypto/x509"
	"github.com/miekg/dns"
	"io/ioutil"
)

func loadRoots(caPath string) *x509.CertPool {
	if caPath == "" {
		return nil
	}

	roots := x509.NewCertPool()
	pem, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil
	}
	ok := roots.AppendCertsFromPEM(pem)
	if !ok {
		return nil
	}
	return roots
}

func loadTlsConfig(cfg TlsConfig) *tls.Config {
	root := loadRoots(cfg.CaPath)
	if cfg.KeyPath == "" || cfg.CertPath == "" {
		return &tls.Config{RootCAs: root}
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: root}
}

func NewServer(config []Config) []*dns.Server {
	var servers []*dns.Server
	for _, cfg := range config {
		if cfg.Count < 1 {
			cfg.Count = 1
		}
		for i := 0; i < cfg.Count; i++ {
			server := &dns.Server{
				Addr:      cfg.Ip + ":" + strconv.Itoa(cfg.Port),
				Net:       cfg.Protocol,
				ReusePort: true,
			}
			if cfg.Tls.Enable {
				server.TLSConfig = loadTlsConfig(cfg.Tls)
			}
			servers = append(servers, server)
		}
	}
	return servers
}
