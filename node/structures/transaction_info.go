package structures

// Sructures to display extra info related to tranactions

type TransactionsHistory struct {
	IOType  bool
	TXID    []byte
	Address string
	Value   float64
}
