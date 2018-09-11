package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"io"
	"math/big"
	"testing"
)

func TestSignatureFormat(t *testing.T) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)

	if err != nil {
		t.Fatalf("Can not make private key %s", err.Error())
	}

	dataToSign := []byte("test data to sign")

	// repeat 1000 times
	for i := 1; i <= 1000; i++ {
		signature, err := SignData(*private, dataToSign)

		if err != nil {
			t.Fatalf("Can not make signature %s", err.Error())
		}

		if len(signature) < 68 || len(signature) > 72 {
			t.Fatalf("Unexpected signature len %d", len(signature))
		}

		// extract r and s and verify signature
		lenR := int(signature[3])
		r := signature[4 : 4+lenR]

		lenS := int(signature[5+lenR])

		s := signature[6+lenR : 6+lenR+lenS]

		ri := big.Int{}
		si := big.Int{}

		ri.SetBytes(r)
		si.SetBytes(s)

		h := sha1.New()
		str := string(dataToSign)
		io.WriteString(h, str)
		data := h.Sum(nil)

		if !ecdsa.Verify(&private.PublicKey, data, &ri, &si) {
			t.Fatal("Signature does not match")
		}

	}
}

func TestVerify(t *testing.T) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)

	if err != nil {
		t.Fatalf("Can not make private key %s", err.Error())
	}

	dataToSign := []byte("test data to sign")

	// repeat 1000 times
	for i := 1; i <= 1000; i++ {
		signature, err := SignData(*private, dataToSign)

		if err != nil {
			t.Fatalf("Can not make signature %s", err.Error())
		}

		pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

		v, err := VerifySignature(signature, dataToSign, pubKey)

		if err != nil {
			t.Fatalf("Verify error: %s", err.Error())
		}

		if !v {
			t.Fatalf("Signature does not match")
		}
	}
}
