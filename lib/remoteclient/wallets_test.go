package remoteclient

import (
	"os"
	"testing"

	"github.com/gelembjuk/oursql/lib/utils"
)

func TestSaveAndLoad(t *testing.T) {

	ws := NewWallets("./")

	defer cleantestFile()

	addresses := []string{}

	for i := 1; i <= 3; i++ {
		addr, err := ws.CreateWallet()

		if err != nil {
			t.Fatalf("Create error: %s", err.Error())
		}

		addresses = append(addresses, addr)
	}

	ws2 := NewWallets("./")

	ws2.LoadFromFile()

	message := "Message to sign"

	signature, err := utils.SignData(ws2.Wallets[addresses[0]].PrivateKey, []byte(message))

	if err != nil {
		t.Fatalf("Signing 1 failed: %s", err.Error())
	}

	vr, err := utils.VerifySignature(signature, []byte(message), ws.Wallets[addresses[0]].PublicKey)

	if err != nil {
		t.Fatalf("Verify 1 failed: %s", err.Error())
	}

	if !vr {
		t.Fatalf("Verify 1 is FALSE. True expected")
	}

}

func cleantestFile() {

	if _, err := os.Stat("./wallet.dat"); !os.IsNotExist(err) {
		os.Remove("./wallet.dat")
	}
}
