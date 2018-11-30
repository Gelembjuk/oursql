package server

import (
	"time"

	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/nodemanager"
)

type changesChecker struct {
	S             *NodeServer
	logger        *utils.LoggerMan
	stopChan      chan bool
	completeChan  chan bool
	ticker        int
	lastCheckTime int64
}

func StartChangesChecker(s *NodeServer) (c *changesChecker) {
	c = &changesChecker{}

	c.logger = s.Logger
	c.S = s

	c.stopChan = make(chan bool)     // to notify routine to stop
	c.completeChan = make(chan bool) // routine to notify it stopped

	c.ticker = 3

	go c.Run()

	return c
}

// Run function to request other nodes for changes regularly
func (c *changesChecker) Run() {
	for {
		c.logger.TraceExt.Printf("Check changes")
		// check if it is time to exit or no
		exit := false

		select {
		case <-c.stopChan:
			exit = true
		default:
		}

		if exit {
			// exit
			break
		}

		if c.ticker > 0 {
			//c.logger.Trace.Printf("Changes Checker ticker value %d", c.ticker)
			time.Sleep(1 * time.Second)
			c.ticker = c.ticker - 1
			continue
		}
		c.logger.TraceExt.Printf("Changes Checker. Go to check state")

		pullResult, err := c.S.Node.GetCommunicationManager().CheckForChangesOnOtherNodes(c.lastCheckTime)

		if err == nil {
			c.processResults(pullResult)
		}

		// decide when to do next check
		if c.S.Node.NodeNet.CheckHadInputConnects() {
			// other nodes can connect to this node. No need to do extra check often
			c.ticker = 180 // try again in 3 minutes
		} else {
			c.ticker = 5 // 5 seconds as it looks like other nodes can not connect to this node
		}

		//c.S.Node.NodeNet.StartNewSessionForInputConnects()
	}
	c.logger.Trace.Printf("Changes Checker Return routine")
	c.completeChan <- true
}
func (c *changesChecker) Stop() error {
	c.logger.Trace.Println("Stop changes checker")

	close(c.stopChan) // notify routine to stop

	c.logger.TraceExt.Println("Wait changes checker routine done")
	// wait when it is stopped
	<-c.completeChan

	close(c.completeChan)

	c.logger.TraceExt.Println("Changes Checker Stopped")

	return nil
}

func (c changesChecker) processResults(res nodemanager.ChangesPullResults) {
	if len(res.AddedTransactions) > 0 {
		// if some transactions were added, notify to build new block
		c.S.blocksMakerObj.DoNewBlock() // this will check state of the pool and start minting new block if there are enough
	}
	if res.AnyChangesPulled() {
		c.logger.Trace.Println("Pull results")
		c.logger.Trace.Printf("Transactions %d:", len(res.AddedTransactions))
		for _, txID := range res.AddedTransactions {
			c.logger.Trace.Printf("   %x", txID)
		}
		c.logger.Trace.Printf("Blocks %d:", len(res.AddedBlocks))
		for _, bHash := range res.AddedBlocks {
			c.logger.Trace.Printf("   %x", bHash)
		}
		c.logger.Trace.Printf("Nodes %d:", len(res.AddedNodes))
		for _, n := range res.AddedNodes {
			c.logger.Trace.Printf("   %s", n)
		}
	}

}
