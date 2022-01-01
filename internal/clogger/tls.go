package clogger

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
)

// TLSConfig is a quick config that contains various
// configs that can be used to construct a TLS server
type TLSConfig struct {
	cert    *tls.Certificate
	caCerts *x509.CertPool
}

func NewTLSConfig(caFile, certPath, keyPath string) (conf TLSConfig, err error) {
	if certPath == "" && keyPath != "" {
		return conf, fmt.Errorf("invalid cert path - expected both key path _and_ cert path")
	} else if certPath != "" && keyPath == "" {
		return conf, fmt.Errorf("invalid key path - expected both key path _and_ cert path")
	} else if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return conf, err
		}

		conf.cert = &cert
	}

	if caFile != "" {
		cert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return conf, err
		}
		conf.caCerts = x509.NewCertPool()
		conf.caCerts.AppendCertsFromPEM(cert)
	}

	return conf, err
}

func NewTLSConfigFromRaw(raw map[string]string) (TLSConfig, error) {
	caFile := raw["cafile"]
	certPath := raw["cert"]
	keyPath := raw["key"]

	return NewTLSConfig(caFile, certPath, keyPath)
}

func (t *TLSConfig) isEnabled() bool {
	return t.cert != nil || t.caCerts != nil
}

// WrapListener returns the given listener wrapped by the TLS config
// If this TLS config is empty, this just returns the given wrapper
func (t *TLSConfig) WrapListener(n net.Listener) net.Listener {
	if !t.isEnabled() {
		return n
	}

	conf := tls.Config{}

	if t.cert != nil {
		conf.Certificates = []tls.Certificate{*t.cert}
	}

	if t.caCerts != nil {
		conf.ClientCAs = t.caCerts
	}

	return tls.NewListener(n, &conf)
}
