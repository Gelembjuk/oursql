package consensus

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/fatih/structs"
	"github.com/gelembjuk/oursql/node/structures"
	"github.com/mitchellh/mapstructure"
)

const (
	KindConseususPoW = "proofofwork"
)

type ConsensusConfigTable struct {
	Table            string
	AllowRowDelete   bool
	AllowRowUpdate   bool
	AllowRowInsert   bool
	AllowTableCreate bool
}

type ConsensusConfig struct {
	ApplicationName   string
	Kind              string
	CoinsForBlockMade float64
	Settings          map[string]interface{}
	AllowTableCreate  bool
	AllowTableDrop    bool
	AllowRowDelete    bool
	UnmanagedTables   []string
	TableRules        []ConsensusConfigTable
	InitNodesAddreses []string
}

// Load config from config file. Some config options an be missed
// missed options must be replaced with default values correctly
func NewConfigFromFile(filepath string) (*ConsensusConfig, error) {
	// we open a file only if it exists. in other case options can be set with command line

	jsonStr, err := ioutil.ReadFile(filepath)

	if err != nil {
		// error is bad only if file exists but we can not open to read
		return nil, err
	}

	config := ConsensusConfig{}

	err = json.Unmarshal(jsonStr, &config)

	if err != nil {
		return nil, err
	}

	if config.CoinsForBlockMade == 0 {
		config.CoinsForBlockMade = 10
	}

	if config.Kind == "" {
		config.Kind = KindConseususPoW
	}
	if config.Kind == KindConseususPoW {
		// check all PoW settings are done
		s := ProofOfWorkSettings{}

		mapstructure.Decode(config.Settings, &s)

		s.completeSettings()

		config.Settings = structs.Map(s)
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
	return &c, nil
}

func (cc ConsensusConfig) GetInfoForTransactions() structures.ConsensusInfo {
	return structures.ConsensusInfo{cc.CoinsForBlockMade}
}

func (cc ConsensusConfig) ExportToFile(filepath string, defaultaddresses string, appname string, thisnodeaddr string) error {
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
		return errors.New("List of default addresses is empty")
	}

	if appname != "" {
		cc.ApplicationName = appname
	}

	if cc.ApplicationName == "" {
		return errors.New("Application name is empty. It is required")
	}

	jsondata, err := json.Marshal(cc)

	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath, jsondata, 0644)

	return err
}
