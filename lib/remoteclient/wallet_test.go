package remoteclient

import (
	"testing"

	"github.com/gelembjuk/oursql/lib/utils"
)

func TestKeysEncoding(t *testing.T) {
	wallet := Wallet{}
	wallet.MakeWallet()

	PubKeyEnc := wallet.GetPublicKeyEncoded()
	PriKeyEnc := wallet.GetPrivateKeyEncoded()

	message := "Message to sign"

	newWallet, err := MakeWalletFromEncoded(PubKeyEnc, PriKeyEnc)

	if err != nil {
		t.Fatalf("Wallet restore failed: %s", err.Error())
	}

	signature, err := utils.SignData(wallet.PrivateKey, []byte(message))

	if err != nil {
		t.Fatalf("Signing 1 failed: %s", err.Error())
	}

	vr, err := utils.VerifySignature(signature, []byte(message), newWallet.PublicKey)

	if err != nil {
		t.Fatalf("Verify 1 failed: %s", err.Error())
	}

	if !vr {
		t.Fatalf("Verify 1 is FALSE. True expected")
	}

	signature, err = utils.SignData(newWallet.PrivateKey, []byte(message))

	if err != nil {
		t.Fatalf("Signing 2 failed: %s", err.Error())
	}

	vr, err = utils.VerifySignature(signature, []byte(message), wallet.PublicKey)

	if err != nil {
		t.Fatalf("Verify 2 failed: %s", err.Error())
	}

	if !vr {
		t.Fatalf("Verify 2 is FALSE. True expected")
	}
}
