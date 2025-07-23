package base

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

func PkToAddress(pk string) string {
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address.String()
}
