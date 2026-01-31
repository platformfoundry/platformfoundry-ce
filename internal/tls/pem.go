package tls

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// encodePEM encodes data to PEM format
func encodePEM(blockType string, data []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  blockType,
		Bytes: data,
	})
}

// decodePEM decodes PEM data
func decodePEM(data []byte) (*pem.Block, []byte) {
	return pem.Decode(data)
}

// saveCertificatePEM saves a certificate to a PEM file
func saveCertificatePEM(cert *x509.Certificate, filename string) error {
	certPEM := encodePEM("CERTIFICATE", cert.Raw)
	return os.WriteFile(filename, certPEM, 0644)
}

// savePrivateKeyPEM saves a private key to a PEM file
func savePrivateKeyPEM(keyBytes []byte, filename string) error {
	keyPEM := encodePEM("PRIVATE KEY", keyBytes)
	return os.WriteFile(filename, keyPEM, 0600)
}

// LoadCertificatePEM loads a certificate from a PEM file
func LoadCertificatePEM(filename string) (*x509.Certificate, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	block, _ := decodePEM(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}
