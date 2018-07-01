package structures

const (
	TXTypeCurrency         = "currency"
	TXTypeCurrencyTransfer = "currencytransfer"
	TXTypeCurrencyCoinbase = "currencycoinbase"
	TXTypeSQL              = "sql"

	txTypeCurrency = 1
	txTypeSQL      = 2
)

// Common interface for all transaction included in a block
type TransactionInterface interface {
	CheckTypeIs(t string) bool
	CheckSubTypeIs(t string) bool

	Verify(prevTXs map[int]TransactionInterface) error
	// return data to make signatures. slice of slices. each slice to sign
	// input transactions must be same type as this transaction. this is controlled outside
	// can be extra constolled in implementation
	PrepareSignData(prevTXs map[int]TransactionInterface) ([][]byte, error)
	SetSignatures(signatures [][]byte) error
	NeedsSignatures() bool
	// returns truc if transacion structure is fully complete. has ID, signatures etc
	IsComplete() bool

	String() string
	Copy() (TransactionInterface, error)
	ToBytes() ([]byte, error)
	serialize() ([]byte, error)

	GetID() []byte
	GetTime() int64
}
