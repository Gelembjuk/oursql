package structures

// CurrencyTransaction represents a Bitcoin transaction
type SQLTransaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
	Time int64
}
