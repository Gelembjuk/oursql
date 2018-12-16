package lib

const ApplicationTitle = "OurSQL"
const ApplicationVersion = "0.1.8 beta"

const Version = byte(0x00)
const AddressChecksumLen = 4

const CurrencySmallestUnit = 0.00000001

const NullAddressString = "NULLADDRESS"

const (
	TXFlagsNothing                    = 0 // 0
	TXFlagsExecute                    = 1 // 1
	TXFlagsNoPool                     = 2 // 2
	TXFlagsSkipSQLBaseCheck           = 4 // 4
	TXFlagsSkipSQLBaseCheckIfNotOnTop = 8 // 8
	TXFlagsNoExecute                  = 16
	TXFlagsVerifyAllowMissed          = 32
	TXFlagsBasedOnTopOfChain          = 64
)
