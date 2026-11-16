package user

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

func LoadPublicKeyPEM(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ECDSA")
	}
	return ec, nil
}

func ResolvePEM(inlinePEM, filePath string) ([]byte, error) {
	if strings.TrimSpace(filePath) != "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read key file %s: %w", filePath, err)
		}
		return b, nil
	}
	s := strings.TrimSpace(inlinePEM)
	if s == "" {
		return nil, fmt.Errorf("key PEM or file path required")
	}
	s = strings.ReplaceAll(s, `\n`, "\n")
	return []byte(s), nil
}
