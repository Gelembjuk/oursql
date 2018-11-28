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

	"github.com/gelembjuk/oursql/lib/dbproxy"
	netlib "github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/nodeclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/nodemanager"
)

type NodeServer struct {
	ConfigDir string
	Node      *nodemanager.Node

	NodeAddress netlib.NodeAddr // Port can be different from NodePort. NodeAddress is address exposed outside
	NodePort    int             // This is the port where a server will listen

	Transit nodeTransit

	Logger *utils.LoggerMan
	// Channels to manipulate roitunes
	StopMainChan        chan struct{}
	StopMainConfirmChan chan struct{}

	changesCheckerObj *changesChecker
	blocksMakerObj    *blocksMaker

	DBProxyAddr string
	DBAddr      string
	QueryFilter *queryFilter

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

	s.Logger.Trace.Printf("Received %s command", command)

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

	case nodeclient.CommandBlock:
		rerr = requestobj.handleBlock()

	case nodeclient.CommandGetBlock:
		rerr = requestobj.handleGetBlock()

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

	case nodeclient.CommandGetBalance:
		rerr = requestobj.handleGetBalance()

	case nodeclient.CommandGetFirstBlocks:
		rerr = requestobj.handleGetFirstBlocks()

	case nodeclient.CommandGetConsensusData:
		rerr = requestobj.handleGetConsensusData()

	case "tx":
		rerr = requestobj.handleTx()

	case "txdata":
		rerr = requestobj.handleTxData()

	case "txcurrequest":
		rerr = requestobj.handleTxCurRequest()

	case "txsqlrequest":
		rerr = requestobj.handleTxSQLRequest()

	case "getnodes":
		rerr = requestobj.handleGetNodes()

	case "addnode":
		rerr = requestobj.handleAddNode()

	case "removenode":
		rerr = requestobj.handleRemoveNode()

	case nodeclient.CommandGetState:
		rerr = requestobj.handleGetState()

	case nodeclient.CommandGetUpdates:
		rerr = requestobj.handleGetUpdates()

	case nodeclient.CommandGetTransaction:
		rerr = requestobj.handleGetTransaction()

	case nodeclient.CommandCheckBlock:
		rerr = requestobj.handleCheckBlock()

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
	s.Logger.Trace.Printf("Prepare server to start %s on a localport %d", s.NodeAddress.NodeAddrToString(), s.NodePort)

	returnWithError := func(err error) error {
		serverStartResult <- err.Error()

		s.stopAllSubroutines()

		s.Logger.Trace.Println("Fail to start server: ", err.Error())

		return err
	}
	// this channel must be inited here. It is used inside StartDatabaseProxy()
	// DB proxy wil notify about new transactions using this channel
	err := s.initBlocksMaker()

	if err != nil {
		return returnWithError(err)
	}

	started, err := s.startDatabaseProxy()

	if err != nil {
		return returnWithError(err)
	}

	if !started {
		s.Logger.Trace.Printf("DB Proxy was not started, was not requested in config")
	}

	// We listen on a port on all interfaces
	ln, err := net.Listen(netlib.Protocol, ":"+strconv.Itoa(s.NodePort))

	if err != nil {
		return returnWithError(err)
	}
	defer ln.Close()

	// client will use the address to include it in requests
	s.Node.NodeClient.SetNodeAddress(s.NodeAddress)

	s.Node.SendVersionToNodes([]netlib.NodeAddr{})

	s.Logger.Trace.Println("Start block bilding routine")

	// we set buffer to 100 transactions.
	// we don't expect more 100 TX will be received while building a block. if yes, we will skip
	// adding a signal. this will not be a problem

	err = s.startChangesChecker()
	if err != nil {
		return returnWithError(err)
	}
	// run blocks maker routine
	err = s.blocksMakerObj.Start()

	if err != nil {
		return returnWithError(err)
	}

	// notify daemon about server started fine
	serverStartResult <- ""

	s.Logger.Trace.Printf("Start listening connections on port %d", s.NodePort)

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

			s.stopAllSubroutines()

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
	s.blocksMakerObj.NewTransaction(tx)
}

// MySQL proxy server. It is in the middle between a DB server and DB client an reads requests
func (s *NodeServer) startDatabaseProxy() (started bool, err error) {

	s.QueryFilter, err = InitQueryFilter(s.DBProxyAddr, s.DBAddr, s.Node.Clone(), s.Logger, s.blocksMakerObj)
	started = true

	if err != nil {
		s.QueryFilter = nil

		if errc, ok := err.(*dbproxy.DBProxyError); ok {

			if errc.IsDBProxyConfigError() {
				return false, nil
			}
		}
		started = false
	}

	return
}

// MySQL proxy server. It is in the middle between a DB server and DB client an reads requests
func (s *NodeServer) startChangesChecker() error {

	s.changesCheckerObj = StartChangesChecker(s)

	return nil
}

// MySQL proxy server. It is in the middle between a DB server and DB client an reads requests
func (s *NodeServer) initBlocksMaker() error {

	s.blocksMakerObj = InitBlocksMaker(s)

	return nil
}

// Stop all sub routines on server stop
func (s *NodeServer) stopAllSubroutines() {
	if s.QueryFilter != nil {
		// MySQL proxy server. It is in the middle between a DB server and DB client an reads requests
		err := s.QueryFilter.Stop()

		if err != nil {
			s.Logger.Error.Printf("Error when stop proxy %s", err.Error())
		}
		s.QueryFilter = nil
	}

	if s.changesCheckerObj != nil {
		s.changesCheckerObj.Stop()
		s.changesCheckerObj = nil
	}

	if s.blocksMakerObj != nil {
		s.blocksMakerObj.Stop()

		s.blocksMakerObj = nil
	}

	// notify daemon process about this server did all completion
	close(s.StopMainConfirmChan)
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
