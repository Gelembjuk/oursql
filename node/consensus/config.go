package consensus

import (
	"encoding/json"
	"os"

	"github.com/fatih/structs"
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
	Kind             string
	Settings         map[string]interface{}
	AllowTableCreate bool
	AllowTableDrop   bool
	AllowRowDelete   bool
	UnmanagedTables  []string
	TableRules       []ConsensusConfigTable
}

func NewConfigFromFile(filepath string) (*ConsensusConfig, error) {
	file, errf := os.Open(filepath)

	if errf != nil {
		// error is bad only if file exists but we can not open to read
		return nil, errf
	}

	config := ConsensusConfig{}
	// we open a file only if it exists. in other case options can be set with command line
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&config)

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func NewConfigDefault() (*ConsensusConfig, error) {
	c := ConsensusConfig{}
	c.Kind = KindConseususPoW
	c.AllowTableCreate = true
	c.AllowTableDrop = true
	c.AllowRowDelete = true
	c.UnmanagedTables = []string{}
	c.TableRules = []ConsensusConfigTable{}

	// make defauls PoW settings
	s := ProofOfWorkSettings{}
	s.completeSettings()

	c.Settings = structs.Map(s)
	return &c, nil
}
