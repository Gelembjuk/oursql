package server

import (
	"github.com/gelembjuk/oursql/lib/utils"
)

type blocksMaker struct {
	S               *NodeServer
	logger          *utils.LoggerMan
	started         bool
	completeChan    chan bool
	blockBilderChan chan []byte

	transactionsInProgress [][]byte
}

func InitBlocksMaker(s *NodeServer) (c *blocksMaker) {
	c = &blocksMaker{}

	c.logger = s.Logger
	c.S = s

	c.started = false

	c.blockBilderChan = make(chan []byte, 100)

	c.transactionsInProgress = [][]byte{}

	return c
}

// Run function to request other nodes for changes regularly
func (c *blocksMaker) Start() error {
	c.completeChan = make(chan bool) // routine to notify it stopped

	go c.run()

	c.started = true
	return nil
}

/*
* The routine that tries to make blocks.
* The routine reads last added transaction ID
* The ID will be real tranaction ID only if this transaction wa new created on this node
* in this case, if block is not created, the transaction will be sent to all other nodes
* it is needed to delay sending of transaction to be able to create a block first, before all other eceive new transaction
* This ID can be also {0} (one byte slice). it means try to create a block but don't send transaction
* and it can be empty slice . it means to exit from teh routibe
 */
func (c *blocksMaker) run() {

	for {
		txID := <-c.blockBilderChan

		c.logger.Trace.Printf("BlockBuilder new transaction %x", txID)

		if len(txID) == 0 {
			// this is return signal from main thread
			close(c.blockBilderChan)
			c.logger.Trace.Printf("Exit BlockBuilder thread")
			break
		}

		// we create separate node object for this thread
		// pointers are used everywhere. so, it can be some sort of conflict with main thread
		NodeClone := c.S.Node.Clone()

		callbackToStoreTransactions := func(list [][]byte) error {
			c.transactionsInProgress = list
			return nil
		}

		// try to buid new block
		_, err := NodeClone.TryToMakeBlock(txID, callbackToStoreTransactions)

		// clean list of locked transactions
		c.transactionsInProgress = [][]byte{}

		if err != nil {
			c.logger.Trace.Printf("Block building error %s\n", err.Error())
		}
	}

	c.logger.Trace.Printf("Block Maker Return routine")
	c.completeChan <- true
}

// New transaction appeared in a pool. Block maker should try to do new block
// if no anough transactions it will just send this TX to all known nodes
func (c *blocksMaker) NewTransaction(tx []byte) {
	// don't block sending. buffer size is 100
	// TX will be skipped if a buffer is full
	select {
	case c.blockBilderChan <- tx: // send signal to block building thread to try to make new block now
	default:
	}
}

// Send command to block builder to start doing new block. It will check if there are eenough transactions
// if no anough it will just continue to wait
func (c *blocksMaker) DoNewBlock() {
	c.NewTransaction([]byte{1})
}

// Returns list of transactions that are currently locked by block building process
// this are transaction taht are still ina pool but we already started to make new block from it
func (c blocksMaker) GetLockedTransactions() [][]byte {
	return c.transactionsInProgress
}

// Stop the routine
func (c *blocksMaker) Stop() error {
	c.logger.Trace.Println("Stop block maker")

	if !c.started {
		// routine was not really started
		close(c.blockBilderChan)
		return nil
	}
	c.blockBilderChan <- []byte{} // send signal to block building thread to exit
	// empty slice means this is exit signal

	// wait when it is stopped
	<-c.completeChan

	close(c.completeChan)

	c.started = false

	return nil
}
