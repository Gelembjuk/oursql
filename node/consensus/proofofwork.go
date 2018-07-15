package consensus

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/config"
	"github.com/gelembjuk/oursql/node/structures"
)

var (
	maxNonce = math.MaxInt64
)

// ProofOfWork represents a proof-of-work
type ProofOfWork struct {
	block  *structures.Block
	target *big.Int
}

// NewProofOfWork builds and returns a ProofOfWork object
// The object can be used to find a hash for the block
func NewProofOfWork(b *structures.Block) *ProofOfWork {
	target := big.NewInt(1)

	var tb int

	if b.Height >= 1000 {
		tb = config.TargetBits_2
	} else {
		tb = config.TargetBits
	}

	target.Lsh(target, uint(256-tb))

	pow := &ProofOfWork{b, target}

	return pow
}

// Prepares data for next iteration of PoW
// this will be hashed
func (pow *ProofOfWork) prepareData() ([]byte, error) {
	txshash, err := pow.block.HashTransactions()

	if err != nil {
		return nil, err
	}

	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			txshash,
			utils.IntToHex(pow.block.Timestamp),
			utils.IntToHex(int64(config.TargetBits)),
		},
		[]byte{},
	)

	return data, nil
}

func (pow *ProofOfWork) addNonceToPrepared(data []byte, nonce int) []byte {
	data = append(data, utils.IntToHex(int64(nonce))...)

	return data
}

// Run performs a proof-of-work
func (pow *ProofOfWork) Run() (int, []byte, error) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	predata, err := pow.prepareData()

	if err != nil {
		return 0, nil, errors.New(fmt.Sprintf("Pow run: %s", err.Error()))
	}

	for nonce < maxNonce {
		// prepare data for next nonce
		data := pow.addNonceToPrepared(predata, nonce)
		// hash
		hash = sha256.Sum256(data)

		// check hash is what we need
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}

	return nonce, hash[:], nil
}

// Validate validates block's PoW
// It calculates hash from same data and check if it is equal to block hash
func (pow *ProofOfWork) Validate() (bool, error) {
	var hashInt big.Int

	predata, err := pow.prepareData()

	if err != nil {
		return false, err
	}

	data := pow.addNonceToPrepared(predata, pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid, nil
}
