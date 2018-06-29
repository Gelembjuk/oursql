package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"io"
	"math/big"
)

func SignData(privKey ecdsa.PrivateKey, dataToSign []byte) ([]byte, error) {
	h := md5.New()
	str := string(dataToSign)
	io.WriteString(h, str)
	data := h.Sum(nil)

	r, s, err := ecdsa.Sign(rand.Reader, &privKey, data)

	if err != nil {
		return nil, err
	}
	signature := append(r.Bytes(), s.Bytes()...)

	return signature, nil
}

func VerifySignature(signature []byte, message []byte, PubKey []byte) (bool, error) {
	h := md5.New()
	str := string(message)
	io.WriteString(h, str)
	data := h.Sum(nil)

	// build key and verify data
	r := big.Int{}
	s := big.Int{}
	sigLen := len(signature)
	r.SetBytes(signature[:(sigLen / 2)])
	s.SetBytes(signature[(sigLen / 2):])

	x := big.Int{}
	y := big.Int{}
	keyLen := len(PubKey)
	x.SetBytes(PubKey[:(keyLen / 2)])
	y.SetBytes(PubKey[(keyLen / 2):])

	curve := elliptic.P256()

	rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

	return ecdsa.Verify(&rawPubKey, data, &r, &s), nil
}

func SignDataSet(PubKey []byte, privKey ecdsa.PrivateKey, dataSetsToSign [][]byte) ([][]byte, error) {
	signatures := [][]byte{}

	for _, dataToSign := range dataSetsToSign {

		attempt := 1

		var signature []byte
		var err error

		for {
			// we can do more 1 attempt to sign. we found some cases where verification of signature fails
			// we don't know the reason
			signature, err = SignData(privKey, dataToSign)

			if err != nil {
				return nil, err
			}

			attempt = attempt + 1

			v, err := VerifySignature(signature, dataToSign, PubKey)

			if err != nil {
				return nil, err
			}

			if v {
				break
			}

			if attempt > 10 {
				break
			}
		}
		signatures = append(signatures, signature)
	}

	return signatures, nil
}
