// Copyright 2021 Hewlett Packard Enterprise Development LP

package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	tokens "github.com/Cray-HPE/spire-tokens/go"
)

type keypair struct {
	certMu   sync.RWMutex
	cert     *tls.Certificate
	certPath string
	keyPath  string
}

// watchCert watches for when the TLS certificate link is updated and then tells
// the http server to use the updated certificate.
// The link is checked to see if it's changed every 5 minutes.
func watchCert(certPath string, result *keypair) {
	// Kubernetes provides the TLS certificate secret as a set of soft links.
	// Due to how these links are updated we cannot use fsNotify.
	// Instead this compares the old path against the current path and
	// updates the certificate when the links change.
	oldPath, err := filepath.EvalSymlinks(certPath)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			currentPath, err := filepath.EvalSymlinks(certPath)
			if err != nil {
				log.Fatal(err)
			}
			if oldPath != currentPath {
				oldPath = currentPath
				log.Printf("Reloading TLS certificate from %q", certPath)
				if err := result.reloadCert(); err != nil {
					log.Printf("Keeping old TLS certificate because the new one could not be loaded: %v", err)
				}
			}
			var waitTime time.Duration = 5 * time.Minute
			time.Sleep(waitTime)
		}
	}()
}

// keypairReloader handles the loading and updating of the TLS certificate
func keypairReloader(certPath, keyPath string) (*keypair, error) {
	result := keypair{
		certPath: certPath,
		keyPath:  keyPath,
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	result.cert = &cert

	go watchCert(certPath, &result)

	return &result, nil
}

// reloadCert updates the cert in the keypair struct to use the latest TLS certificate
func (k *keypair) reloadCert() error {
	newCert, err := tls.LoadX509KeyPair(k.certPath, k.keyPath)
	if err != nil {
		return err
	}
	k.certMu.Lock()
	defer k.certMu.Unlock()
	k.cert = &newCert
	return nil
}

// GetCerGetCertificateFunc is a custom GetCertificate function that is used to
// handle the fetching of the TLS cerficate in the http server
func (k *keypair) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		k.certMu.RLock()
		defer k.certMu.RUnlock()
		return k.cert, nil
	}
}

func main() {
	log.Printf("Server started")

	tlsCertPath := "/tls/tls.crt"
	tlsKeyPath := "/tls/tls.key"

	router := tokens.NewRouter()

	keypair, err := keypairReloader(tlsCertPath, tlsKeyPath)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{Addr: ":54440",
		Handler: router,
		TLSConfig: &tls.Config{
			GetCertificate: keypair.GetCertificateFunc()}}

	log.Fatal(srv.ListenAndServeTLS("", ""))
}
