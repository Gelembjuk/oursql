package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"io"
	"math/big"
)

func SignData(privKey ecdsa.PrivateKey, dataToSign []byte) ([]byte, error) {
	h := sha1.New()
	str := string(dataToSign)
	io.WriteString(h, str)
	data := h.Sum(nil)

	r, s, err := ecdsa.Sign(rand.Reader, &privKey, data)

	if err != nil {
		return nil, err
	}

	return signatureToDER(r.Bytes(), s.Bytes()), nil
}

// Make DER format signature from r and s slices
func signatureToDER(r, s []byte) []byte {
	sig := []byte{}

	sig = append(sig, 0x02)
	sig = append(sig, uint8(len(r)))
	sig = append(sig, r...)

	sig = append(sig, 0x02)
	sig = append(sig, uint8(len(s)))
	sig = append(sig, s...)

	allsig := []byte{0x30, uint8(len(sig))}

	allsig = append(allsig, sig...)

	return allsig
}

// Get R and S from DER formatted signature
func signatureFromDER(dersig []byte) (r, s []byte, err error) {
	if len(dersig) < 4 {
		err = errors.New("Wrong signature length")
		return
	}

	lenR := int(dersig[3])

	if len(dersig) < 7+lenR {
		err = errors.New("Wrong signature length")
		return
	}

	r = dersig[4 : 4+lenR]

	lenS := int(dersig[5+lenR])

	if len(dersig) < 6+lenR+lenS {
		err = errors.New("Wrong signature length")
		return
	}

	s = dersig[6+lenR : 6+lenR+lenS]

	return
}
func VerifySignature(signature []byte, message []byte, PubKey []byte) (bool, error) {
	h := sha1.New()
	str := string(message)
	io.WriteString(h, str)
	data := h.Sum(nil)

	sigR, sigS, err := signatureFromDER(signature)

	if err != nil {
		return false, err
	}

	r := big.Int{}
	s := big.Int{}

	r.SetBytes(sigR)
	s.SetBytes(sigS)

	// build key and verify data
	x := big.Int{}
	y := big.Int{}
	keyLen := len(PubKey)
	x.SetBytes(PubKey[:(keyLen / 2)])
	y.SetBytes(PubKey[(keyLen / 2):])

	curve := elliptic.P256()

	rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

	return ecdsa.Verify(&rawPubKey, data, &r, &s), nil
}
func SignDataByPubKey(PubKey []byte, privKey ecdsa.PrivateKey, dataToSign []byte) ([]byte, error) {
	var signature []byte
	var err error

	signature, err = SignData(privKey, dataToSign)

	if err != nil {
		return nil, err
	}

	v, err := VerifySignature(signature, dataToSign, PubKey)

	if err != nil {
		return nil, err
	}

	if !v {
		return nil, errors.New("Just created signature looks wrong!")
	}

	return signature, nil
}
func SignDataSet(PubKey []byte, privKey ecdsa.PrivateKey, dataSetsToSign [][]byte) ([][]byte, error) {
	signatures := [][]byte{}

	for _, dataToSign := range dataSetsToSign {

		var signature []byte
		var err error

		for {
			signature, err = SignData(privKey, dataToSign)

			if err != nil {
				return nil, err
			}

			v, err := VerifySignature(signature, dataToSign, PubKey)

			if err != nil {
				return nil, err
			}

			if !v {
				return nil, errors.New("Just created signature looks wrong!")
			}
		}

		signatures = append(signatures, signature)
	}

	return signatures, nil
}
