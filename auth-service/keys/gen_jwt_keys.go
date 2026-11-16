//go:build ignore

// Generate ES256 P-256 keypair for local/dev.
// Usage: go run ./keys/gen_jwt_keys.go
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir := filepath.Dir(os.Args[0])
	if cwd, err := os.Getwd(); err == nil {
		// prefer keys/ under module when run as go run ./keys/gen_jwt_keys.go
		if _, err := os.Stat(filepath.Join(cwd, "keys")); err == nil {
			dir = filepath.Join(cwd, "keys")
		} else if filepath.Base(cwd) == "keys" {
			dir = cwd
		}
	}
	_ = os.MkdirAll(dir, 0o755)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	privPath := filepath.Join(dir, "jwt-es256-dev.pem")
	pubPath := filepath.Join(dir, "jwt-es256-dev.pub.pem")
	if err := os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", privPath)
	fmt.Println("wrote", pubPath)
}
