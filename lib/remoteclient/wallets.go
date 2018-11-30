package remoteclient

import (
	"encoding/json"
	"errors"
	"fmt"
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

type WalletsFileRec struct {
	Address    string
	PubKey     string
	PrivateKey string
}
type WalletsFile struct {
	Wallets []WalletsFileRec
}

func NewWallets(confdir string) Wallets {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	wallets.ConfigDir = confdir
	return wallets
}

// CreateWallet adds a Wallet to Wallets
func (ws *Wallets) CreateWallet() (string, error) {
	wallet := Wallet{}
	wallet.MakeWallet()

	//address := hex.EncodeToString(wallet.GetAddress())
	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = &wallet

	err := ws.SaveToFile()

	if err != nil {
		return "", err
	}

	return address, nil
}

// Import wallets from external file
func (ws *Wallets) ImportWallet(filepath string) ([]string, error) {
	if filepath == "" {
		return nil, errors.New("File path is empty")
	}
	extwallets := NewWallets("")
	extwallets.Logger = ws.Logger
	extwallets.WalletsFile = filepath

	err := extwallets.LoadFromFile()

	if err != nil {
		return nil, err
	}

	addresses := []string{}

	for addr, w := range extwallets.Wallets {
		if _, ok := ws.Wallets[addr]; ok {
			continue
		}
		ws.Wallets[addr] = w
		addresses = append(addresses, addr)
	}

	err = ws.SaveToFile()

	if err != nil {
		return nil, err
	}

	return addresses, nil
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
	file, errf := os.Open(walletsFile)

	if errf != nil && !os.IsNotExist(errf) {
		return errf
	}
	if errf != nil {
		// wallets file not found
		return nil
	}

	wsc := WalletsFile{}
	// we open a file only if it exists. in other case options can be set with command line
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&wsc)

	if err != nil {
		return err
	}
	for _, w := range wsc.Wallets {

		wallet, err := MakeWalletFromEncoded(w.PubKey, w.PrivateKey)

		if err != nil {
			return err
		}

		ws.Wallets[w.Address] = &wallet
	}

	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveToFile() error {

	walletsFile := ws.ConfigDir + walletFile

	wsc := WalletsFile{}
	wsc.Wallets = []WalletsFileRec{}

	for _, wallet := range ws.Wallets {
		w := WalletsFileRec{string(wallet.GetAddress()), wallet.GetPublicKeyEncoded(), wallet.GetPrivateKeyEncoded()}
		wsc.Wallets = append(wsc.Wallets, w)
	}

	file, errf := os.OpenFile(walletsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if errf != nil {
		return errf
	}

	encoder := json.NewEncoder(file)

	err := encoder.Encode(&wsc)

	if err != nil {
		return err
	}

	file.Close()

	return nil
}
