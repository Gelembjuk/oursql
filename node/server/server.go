package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	netlib "github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/nodeclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/nodemanager"
)

type NodeServer struct {
	ConfigDir string
	Node      *nodemanager.Node

	NodeAddress netlib.NodeAddr

	Transit nodeTransit

	Logger *utils.LoggerMan
	// Channels to manipulate roitunes
	StopMainChan        chan struct{}
	StopMainConfirmChan chan struct{}
	BlockBilderChan     chan []byte

	NodeAuthStr string
}

func (s *NodeServer) GetClient() *nodeclient.NodeClient {

	return s.Node.NodeClient
}

// handle received data. It can be one way command or a request for some data

func (s *NodeServer) handleConnection(conn net.Conn) {
	starttime := time.Now().UnixNano()
	sessid := utils.RandString(5)

	//s.Logger.Trace.Printf("New command. Start reading %s", sessid)

	command, request, authstring, err := s.readRequest(conn)

	if err != nil {
		s.sendErrorBack(conn, errors.New("Network Data Reading Error: "+err.Error()))
		conn.Close()
		return
	}

	s.Logger.Trace.Printf("Received %s command, %s, old sess %s", command, sessid, s.Node.SessionID)

	requestobj := NodeServerRequest{}
	requestobj.Node = s.Node.Clone()
	requestobj.Node.SessionID = sessid
	requestobj.Logger = s.Logger
	requestobj.Request = request[:]
	requestobj.NodeAuthStrIsGood = (s.NodeAuthStr == authstring && len(authstring) > 0)
	requestobj.S = s
	requestobj.S.Node.SessionID = sessid
	requestobj.SessID = sessid

	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		requestobj.RequestIP = addr.IP.String()
	}
	request = nil

	// open blockchain. and close in the end ofthis function
	err = requestobj.Node.DBConn.OpenConnection(sessid)

	if err != nil {
		s.sendErrorBack(conn, errors.New("Blockchain open Error: "+err.Error()))
		conn.Close()
		return
	}

	//s.Logger.Trace.Printf("Nodes Network State: %d , %s", len(requestobj.Node.NodeNet.Nodes), requestobj.Node.NodeNet.Nodes)

	var rerr error

	switch command {
	case "addr":
		rerr = requestobj.handleAddr()
	case "viod":
		// do nothing
		s.Logger.Trace.Println("Void command reveived")
	case "block":
		rerr = requestobj.handleBlock()
	case "inv":
		rerr = requestobj.handleInv()
	case "getblocks":
		rerr = requestobj.handleGetBlocks()

	case "getblocksup":
		rerr = requestobj.handleGetBlocksUpper()

	case "getdata":
		rerr = requestobj.handleGetData()

	case "getunspent":
		rerr = requestobj.handleGetUnspent()

	case "gethistory":
		rerr = requestobj.handleGetHistory()

	case "getbalance":
		rerr = requestobj.handleGetBalance()

	case "getfblocks":
		rerr = requestobj.handleGetFirstBlocks()

	case "tx":
		rerr = requestobj.handleTx()

	case "txdata":
		rerr = requestobj.handleTxData()

	case "txcurrequest":
		rerr = requestobj.handleTxCurRequest()

	case "getnodes":
		rerr = requestobj.handleGetNodes()

	case "addnode":
		rerr = requestobj.handleAddNode()

	case "removenode":
		rerr = requestobj.handleRemoveNode()

	case "getstate":
		rerr = requestobj.handleGetState()

	case "version":
		rerr = requestobj.handleVersion()
	default:
		rerr = errors.New("Unknown command!")
	}

	requestobj.Node.DBConn.CloseConnection()

	if rerr != nil {
		s.Logger.Error.Println("Network Command Handle Error: ", rerr.Error())
		s.Logger.Trace.Println("Network Command Handle Error: ", rerr.Error())

		if requestobj.HasResponse {
			// return error to the client
			// first byte is bool false to indicate there was error
			s.sendErrorBack(conn, rerr)
		}
	}

	if requestobj.HasResponse && requestobj.Response != nil && rerr == nil {
		// send this response back
		// first byte is bool true to indicate request was success
		dataresponse := append([]byte{1}, requestobj.Response...)

		s.Logger.Trace.Printf("Responding %d bytes\n", len(dataresponse))

		_, err := conn.Write(dataresponse)

		if err != nil {
			s.Logger.Error.Println("Sending response error: ", err.Error())
		}
	}
	duration := time.Since(time.Unix(0, starttime))
	ms := duration.Nanoseconds() / int64(time.Millisecond)
	s.Logger.Trace.Printf("Complete processing %s command. Time: %d ms, sess %s", command, ms, sessid)

	conn.Close()
}

// response error to a client
func (s *NodeServer) sendErrorBack(conn net.Conn, err error) {
	s.Logger.Error.Println("Sending back error message: ", err.Error())
	s.Logger.Trace.Println("Sending back error message: ", err.Error())

	payload, err := netlib.GobEncode(err.Error())

	if err == nil {
		dataresponse := append([]byte{0}, payload...)

		s.Logger.Trace.Printf("Responding %d bytes as error message\n", len(dataresponse))

		_, err = conn.Write(dataresponse)

		if err != nil {
			s.Logger.Error.Println("Sending response error: ", err.Error())
		}
	}
}

// Starts a server for node. It listens TPC port and communicates with other nodes and lite clients

func (s *NodeServer) StartServer(serverStartResult chan string) error {
	s.Logger.Trace.Println("Prepare server to start ", s.NodeAddress.NodeAddrToString())

	ln, err := net.Listen(netlib.Protocol, ":"+strconv.Itoa(s.NodeAddress.Port))

	if err != nil {
		serverStartResult <- err.Error()

		close(s.StopMainConfirmChan)
		s.Logger.Trace.Println("Fail to start port listening ", err.Error())
		return err
	}
	defer ln.Close()

	// client will use the address to include it in requests
	s.Node.NodeClient.SetNodeAddress(s.NodeAddress)

	s.Node.SendVersionToNodes([]netlib.NodeAddr{})

	s.Logger.Trace.Println("Start block bilding routine")
	s.BlockBilderChan = make(chan []byte, 100)
	// we set buffer to 100 transactions.
	// we don't expect more 100 TX will be received while building a block. if yes, we will skip
	// adding a signal. this will not be a problem

	// notify daemon about server started fine
	serverStartResult <- ""

	go s.BlockBuilder()

	s.Logger.Trace.Println("Start listening connections on port ", s.NodeAddress.Port)

	for {
		conn, err := ln.Accept()

		if err != nil {
			return err
		}
		// check if is a time to stop this loop
		stop := false

		// check if a channel is still open. It can be closed in agoroutine when receive external stop signal
		select {
		case _, ok := <-s.StopMainChan:

			if !ok {
				stop = true
			}
		default:
		}

		if stop {

			// complete all tasks. save data if needed
			ln.Close()

			close(s.StopMainConfirmChan)

			s.BlockBilderChan <- []byte{} // send signal to block building thread to exit
			// empty slice means this is exit signal

			s.Logger.Trace.Println("Stop Listing Network. Correct exit")
			break
		}

		go s.handleConnection(conn)
	}
	return nil
}

/*
* Sends signal to routine where we make blocks. This makes the routine to check transactions in unapproved cache
* And try to make a block if there are enough transactions
 */
func (s *NodeServer) TryToMakeNewBlock(tx []byte) {
	// don't block sending. buffer size is 100
	// TX will be skipped if a buffer is full
	select {
	case s.BlockBilderChan <- tx: // send signal to block building thread to try to make new block now
	default:
	}
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
func (s *NodeServer) BlockBuilder() {
	for {
		txID := <-s.BlockBilderChan

		s.Logger.Trace.Printf("BlockBuilder new transaction %x", txID)

		if len(txID) == 0 {
			// this is return signal from main thread
			close(s.BlockBilderChan)
			s.Logger.Trace.Printf("Exit BlockBuilder thread")
			return
		}

		// we create separate node object for this thread
		// pointers are used everywhere. so, it can be some sort of conflict with main thread
		NodeClone := s.Node.Clone()
		// try to buid new block
		_, err := NodeClone.TryToMakeBlock(txID)

		if err != nil {
			s.Logger.Trace.Printf("Block building error %s\n", err.Error())
		}

		s.Logger.Trace.Printf("Attempt finished")
	}
}

// Reads and parses request from network data
func (s *NodeServer) readRequest(conn net.Conn) (string, []byte, string, error) {
	// 1. Read command
	commandbuffer, err := s.readFromConnection(conn, netlib.CommandLength)

	if err != nil {
		return "", nil, "", err
	}

	command := netlib.BytesToCommand(commandbuffer)

	// 2. Get length of command data

	lengthbuffer, err := s.readFromConnection(conn, 4)

	if err != nil {
		return "", nil, "", err
	}

	var datalength uint32
	binary.Read(bytes.NewReader(lengthbuffer), binary.LittleEndian, &datalength)

	// 3. Get length of extra data
	lengthbuffer, err = s.readFromConnection(conn, 4)

	if err != nil {
		return "", nil, "", err
	}

	var extradatalength uint32
	binary.Read(bytes.NewReader(lengthbuffer), binary.LittleEndian, &extradatalength)

	// 4. read command data by length
	//s.Logger.Trace.Printf("Before read data %d bytes", datalength)

	databuffer := []byte{}

	if datalength > 0 {
		databuffer, err = s.readFromConnection(conn, int(datalength))

		if err != nil {
			return "", nil, "", errors.New(fmt.Sprintf("Error reading %d bytes of request: %s", datalength, err.Error()))
		}
	}

	// 5. read extra data by length

	authstr := ""

	if extradatalength > 0 {
		extradatabuffer, err := s.readFromConnection(conn, int(extradatalength))

		if err != nil {
			return "", nil, "", errors.New(fmt.Sprintf("Error reading %d bytes of extra data: %s", extradatalength, err.Error()))
		}

		authstr = netlib.BytesToCommand(extradatabuffer)
	}

	return command, databuffer, authstr, nil
}

// Read given amount of bytes from connection
func (s *NodeServer) readFromConnection(conn net.Conn, countofbytes int) ([]byte, error) {
	buff := new(bytes.Buffer)

	pauses := 0

	for {

		tmpbuffer := make([]byte, countofbytes-buff.Len())

		read, err := conn.Read(tmpbuffer)

		if read > 0 {
			buff.Write(tmpbuffer[:read])

			if buff.Len() == countofbytes {
				break
			}
			pauses = 0
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if read == 0 {
			if pauses > 30 {
				break
			}
			time.Sleep(1 * time.Second)
			pauses++
		}
	}

	if buff.Len() < countofbytes {
		return nil,
			errors.New(fmt.Sprintf("Wrong number of bytes received for a request. Expected - %d, read - %d", countofbytes, buff.Len()))
	}

	return buff.Bytes(), nil
}
