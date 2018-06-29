package remoteclient

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gelembjuk/oursql/lib/utils"
)

// Wallets stores a collection of wallets
type Wallets struct {
	ConfigDir string

	Wallets map[string]*Wallet

	Logger *utils.LoggerMan

	WalletsFile string
}

type WalletsFile struct {
	Wallets map[string]*Wallet
}

// CreateWallet adds a Wallet to Wallets
func (ws *Wallets) CreateWallet() (string, error) {
	wallet := Wallet{}
	wallet.MakeWallet()

	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = &wallet

	err := ws.SaveToFile()

	if err != nil {
		return "", err
	}

	return address, nil
}

// GetAddresses returns an array of addresses stored in the wallet file
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// GetWallet returns a Wallet by its address
func (ws Wallets) GetWallet(address string) (Wallet, error) {
	if _, ok := ws.Wallets[address]; ok {
		return *ws.Wallets[address], nil
	}
	return Wallet{}, errors.New("Wallet nout found")
}

// LoadFromFile loads wallets from the file
func (ws *Wallets) LoadFromFile() error {
	var walletsFile string

	if ws.WalletsFile != "" {
		walletsFile = ws.WalletsFile
	} else {
		walletsFile = ws.ConfigDir + walletFile
	}

	_, err := os.Stat(walletsFile)

	if err != nil {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletsFile)
	if err != nil {
		return err
	}

	var wallets WalletsFile
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {

		return err
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveToFile() error {
	var content bytes.Buffer
	walletsFile := ws.ConfigDir + walletFile

	gob.Register(elliptic.P256())

	wsc := WalletsFile{}
	wsc.Wallets = ws.Wallets

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(wsc)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(walletsFile, content.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
