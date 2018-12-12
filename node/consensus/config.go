package consensus

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/dbquery"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/mitchellh/mapstructure"
)

const (
	KindConseususPoW = "proofofwork"
)

type ConsensusConfigCost struct {
	Default         float64
	RowDelete       float64
	RowUpdate       float64
	RowInsert       float64
	TableCreate     float64
	ApplyAfterBlock int
}
type ConsensusConfigTable struct {
	Table            string
	AllowRowDelete   bool
	AllowRowUpdate   bool
	AllowRowInsert   bool
	AllowTableCreate bool
	TransactionCost  ConsensusConfigCost
	ApplyAfterBlock  int
}
type ConsensusConfigApplication struct {
	Name    string
	WebSite string
	Team    string
}
type consensusConfigState struct {
	isDefault bool
	filePath  string
}
type ConsensusConfig struct {
	Application            ConsensusConfigApplication
	Kind                   string
	CoinsForBlockMade      float64
	Settings               map[string]interface{}
	ApplyRulesAfterBlock   int
	AllowTableCreate       bool
	AllowTableDrop         bool
	AllowRowDelete         bool
	TransactionCost        ConsensusConfigCost
	UnmanagedTables        []string
	TableRules             []ConsensusConfigTable
	InitNodesAddreses      []string
	PaidTransactionsWallet string
	state                  consensusConfigState
}

// Load config from config file. Some config options an be missed
// missed options must be replaced with default values correctly
func NewConfigFromFile(filepath string) (*ConsensusConfig, error) {
	config := ConsensusConfig{}

	err := config.loadFromFile(filepath)

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func NewConfigDefault() (*ConsensusConfig, error) {
	c := ConsensusConfig{}
	c.Kind = KindConseususPoW
	c.CoinsForBlockMade = 10
	c.AllowTableCreate = true
	c.AllowTableDrop = true
	c.AllowRowDelete = true
	c.UnmanagedTables = []string{}
	c.TableRules = []ConsensusConfigTable{}
	c.InitNodesAddreses = []string{}

	// make defauls PoW settings
	s := ProofOfWorkSettings{}
	s.completeSettings()

	c.Settings = structs.Map(s)

	c.state.isDefault = true
	c.state.filePath = ""

	return &c, nil
}

// Load consensus config JSON from file and parse
func (c *ConsensusConfig) loadFromFile(filepath string) error {

	jsonStr, err := ioutil.ReadFile(filepath)

	if err != nil {
		// error is bad only if file exists but we can not open to read
		return err
	}

	err = c.load(jsonStr)

	if err != nil {
		return err
	}
	c.state.isDefault = false
	c.state.filePath = filepath

	return nil
}

func (c *ConsensusConfig) load(jsonStr []byte) error {

	err := json.Unmarshal(jsonStr, c)

	if err != nil {

		return err
	}

	if c.CoinsForBlockMade == 0 {
		c.CoinsForBlockMade = 10
	}

	if c.Kind == "" {
		c.Kind = KindConseususPoW
	}
	if c.Kind == KindConseususPoW {
		// check all PoW settings are done
		s := ProofOfWorkSettings{}

		mapstructure.Decode(c.Settings, &s)

		s.completeSettings()

		c.Settings = structs.Map(s)
	}

	return nil
}

// Save a consensus config to same file from where it was loaded
func (cc ConsensusConfig) saveBackToFile() error {
	jsondata, err := json.Marshal(cc)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(cc.state.filePath, jsondata, 0644)
}

// Return info about transaction settings
func (cc ConsensusConfig) GetInfoForTransactions() structures.ConsensusInfo {
	return structures.ConsensusInfo{cc.CoinsForBlockMade}
}

// Exports config to file
func (cc ConsensusConfig) ExportToFile(filepath string, defaultaddresses string, appname string, thisnodeaddr string) error {
	jsondata, err := cc.Export(defaultaddresses, appname, thisnodeaddr)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath, jsondata, 0644)

	return err
}

// Exports config to JSON string
func (cc ConsensusConfig) Export(defaultaddresses string, appname string, thisnodeaddr string) (jsondata []byte, err error) {
	addresses := []string{}

	if defaultaddresses != "" {
		list := strings.Split(defaultaddresses, ",")

		for _, a := range list {
			if a == "" {
				continue
			}
			if a == "own" {
				if thisnodeaddr != "" {
					a = thisnodeaddr
				} else {
					continue
				}
			}
			addresses = append(addresses, a)
		}
	}

	if len(addresses) > 0 {
		cc.InitNodesAddreses = addresses
	}

	if len(cc.InitNodesAddreses) == 0 && thisnodeaddr != "" {
		cc.InitNodesAddreses = []string{thisnodeaddr}
	}

	if len(cc.InitNodesAddreses) == 0 {
		err = errors.New("List of default addresses is empty")
		return
	}

	if appname != "" {
		cc.Application.Name = appname
	}

	if cc.Application.Name == "" {
		err = errors.New("Application name is empty. It is required")
		return
	}

	jsondata, err = json.Marshal(cc)

	return
}

// Returns one of addresses listed in initial addresses
func (cc ConsensusConfig) GetRandomInitialAddress() *net.NodeAddr {
	if len(cc.InitNodesAddreses) == 0 {
		return nil
	}
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	addr := cc.InitNodesAddreses[rand.Intn(len(cc.InitNodesAddreses))]

	na := net.NodeAddr{}
	na.LoadFromString(addr)

	return &na
}

// Checks if a config structure was loaded from file or not
func (cc ConsensusConfig) IsDefault() bool {

	return cc.state.isDefault
}

// Set config file path. this defines a path where a config file should be, even if it is not yet here
func (cc *ConsensusConfig) SetConfigFilePath(fp string) {
	cc.state.filePath = fp
}

// Replace consensus config file . It checks if a config is correct, if can be parsed

func (cc *ConsensusConfig) UpdateConfig(jsondoc []byte) error {

	if cc.state.filePath == "" {
		return errors.New("COnfig file path missed. Can not save")
	}

	c := ConsensusConfig{}

	err := c.load(jsondoc)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(cc.state.filePath, jsondoc, 0644)

	if err != nil {
		return err
	}

	// load this just saved contents file
	return cc.loadFromFile(cc.state.filePath)
}

// Returns wallet where to send money spent on paid transactions
func (cc ConsensusConfig) GetPaidTransactionsWallet() string {
	if cc.PaidTransactionsWallet == "" {
		return ""
	}

	pubKeyHash, err := utils.AddresToPubKeyHash(cc.PaidTransactionsWallet)

	if err != nil || len(pubKeyHash) == 0 {
		return ""
	}
	addr, err := utils.PubKeyHashToAddres(pubKeyHash)

	if err != nil {
		return ""
	}
	return addr

}

// Returns wallet where to send money spent on paid transactions
func (cc ConsensusConfig) GetPaidTransactionsWalletPubKeyHash() []byte {
	if cc.PaidTransactionsWallet == "" {
		return []byte{}
	}
	pubKeyHash, err := utils.AddresToPubKeyHash(cc.PaidTransactionsWallet)

	if err != nil || len(pubKeyHash) == 0 {
		return []byte{}
	}

	return pubKeyHash
}

// check custom rule for the table about permissions
func (cc ConsensusConfig) getTableCustomConfig(qp *dbquery.QueryParsed) *ConsensusConfigTable {

	if !qp.IsUpdate() {
		return nil
	}

	if cc.TableRules == nil {
		// no any rules
		return nil
	}

	for _, t := range cc.TableRules {
		if t.Table != qp.Structure.GetTable() {
			continue
		}
		return &t
	}

	return nil
}

// Increase rule start block heigh for all rules
// It is used for initial DB import and create BC on existent data
func (cc *ConsensusConfig) ExtendRulesApplyStartHeigh(setHeigh int) {
	hadchange := false

	if cc.ApplyRulesAfterBlock < setHeigh {
		cc.ApplyRulesAfterBlock = setHeigh
		hadchange = true
	}
	if cc.TransactionCost.ApplyAfterBlock < setHeigh && cc.TransactionCost.hasAnyNonDefaut() {
		cc.TransactionCost.ApplyAfterBlock = setHeigh
		hadchange = true
	}
	for i, _ := range cc.TableRules {
		if cc.TableRules[i].ApplyAfterBlock < setHeigh {
			cc.TableRules[i].ApplyAfterBlock = setHeigh
			hadchange = true
		}

		if cc.TableRules[i].TransactionCost.hasAnyNonDefaut() &&
			cc.TableRules[i].TransactionCost.ApplyAfterBlock < setHeigh {
			cc.TableRules[i].TransactionCost.ApplyAfterBlock = setHeigh
			hadchange = true
		}
	}

	if hadchange {
		cc.saveBackToFile()
	}
}

// Returns trus if a Const structure has any values more 0. False if no any payments required
func (ccc ConsensusConfigCost) hasAnyNonDefaut() bool {
	if ccc.ApplyAfterBlock > 0 {
		return true
	}
	if ccc.Default > 0 {
		return true
	}
	if ccc.RowDelete > 0 {
		return true
	}
	if ccc.RowInsert > 0 {
		return true
	}
	if ccc.RowUpdate > 0 {
		return true
	}
	if ccc.TableCreate > 0 {
		return true
	}
	return false
}
