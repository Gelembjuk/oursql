package server

/*
Error codes
Errors returned by the proxy must have MySQL codes.
2 - Query requires public key
3 - Query requires data to sign
4 - Error preparing of query parsing

*/
import (
	"time"

	"github.com/gelembjuk/oursql/lib/utils"
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
		c.logger.Trace.Printf("Changes Checker. Go to check state")

		c.S.Node.GetCommunicationManager().CheckForChangesOnOtherNodes(c.lastCheckTime)

		// decide when to do next check
		if c.S.hadOtherNodesConnects {
			// other nodes can connect to this node. No need to do extra check often
			c.ticker = 180 // try again in 3 minutes
		} else {
			c.ticker = 5 // 5 seconds as it looks like other nodes can not connect to this node
		}

		c.S.hadOtherNodesConnects = false
	}
	c.logger.Trace.Printf("Changes Checker Return routine")
	c.completeChan <- true
}
func (c *changesChecker) Stop() error {
	c.logger.Trace.Println("Stop changes checker")

	close(c.stopChan) // notify routine to stop

	// wait when it is stopped
	<-c.completeChan

	close(c.completeChan)

	return nil
}
