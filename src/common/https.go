// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Taken and modified from GO website

package common
import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"time"
	"os"
	"net/http"
	"crypto/tls"
)
//fix and set all of these variables

func GenerateCertificate(host string) (key, certificate [] byte) {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
		return
	}
    notBefore := time.Now()
	notAfter := notBefore.Add(14*24*time.Hour)

	template := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
		return
	}
	certBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBlock := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return certBlock,keyBlock
}


func CustomListenAndServeTLS(d *http.ServeMux) error {
    cert,key := GenerateCertificate(os.Getenv("Host"))
    server := &http.Server{Addr: ":900", Handler: d}
    config := &tls.Config{}
    *config = *server.TLSConfig
    if config.NextProtos == nil {
        config.NextProtos = []string{"http/1.1"}
    }
    var err error
    config.Certificates = make([]tls.Certificate, 1)
    config.Certificates[0], err = tls.X509KeyPair(cert,key)
    if err != nil {
        return err
       }
    conn, err := net.Listen("tcp",server.Addr)
    if err != nil {
        return err
    }
    
    tlsListener := tls.NewListener(conn,config)
    return server.Serve(tlsListener)
}
