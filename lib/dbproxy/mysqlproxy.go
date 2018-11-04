package dbproxy

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
)

// proxy implements server for capturing and forwarding MySQL traffic.
type mysqlProxy struct {
	mysqlHost        string
	proxyHost        string
	requestCallback  RequestQueryFilterCallback
	responseCallback ResponseFilterCallback
	queryFilter      DBProxyFilter
	traceLog         *log.Logger
	errorLog         *log.Logger
	state            byte
	stopChan         chan bool
	completeChan     chan bool
}

func NewMySQLProxy(proxyHost, mysqlHost string) (DBProxyInterface, error) {
	obj := &mysqlProxy{}

	if proxyHost == "" {
		return nil, errors.New("DB Proxy listening address is empty. Expected `host:port` or `:port` value")
	}
	if mysqlHost == "" {
		return nil, errors.New("MySQL server host/socker info is missed")
	}

	obj.mysqlHost = mysqlHost
	obj.proxyHost = proxyHost

	obj.SetLoggers(log.New(ioutil.Discard, "", 0),
		log.New(ioutil.Discard, "", 0))
	return obj, nil
}

// Init the object.
func (p *mysqlProxy) Init() error {
	// check if everything is ready
	if p.traceLog == nil {
		return errors.New("No trace logger object!")
	}

	if p.errorLog == nil {
		return errors.New("No error logger object!")
	}
	p.stopChan = make(chan bool)
	p.completeChan = make(chan bool)

	return nil
}

// Set filtering callback function
func (p *mysqlProxy) SetCallbacks(requestCallback RequestQueryFilterCallback, responseCallback ResponseFilterCallback) {
	p.requestCallback = requestCallback
	p.responseCallback = responseCallback
}

// Set query filter structure
func (p *mysqlProxy) SetFilter(filterObj DBProxyFilter) {
	p.queryFilter = filterObj
}

func (p *mysqlProxy) SetLoggers(t *log.Logger, e *log.Logger) {
	p.traceLog = t
	p.errorLog = e
}

// run starts accepting TCP connection and forwarding it to MySQL server.
// Each incoming TCP connection is handled in own goroutine.
func (p *mysqlProxy) Run() error {

	listener, err := net.Listen("tcp", p.proxyHost)

	if err != nil {
		return err
	}

	p.traceLog.Printf("Started listening on %s", p.proxyHost)

	if p.state == 2 {
		return errors.New("Got signal to stop before normal start")
	}

	// state before to start listener
	p.state = 1

	go func() {
		defer listener.Close()

		p.traceLog.Printf("Start routine")

		// listener blocked
		p.state = 2

		for {
			client, err := listener.Accept()

			exit := false

			select {
			case <-p.stopChan:
				exit = true
			default:
			}

			if exit {
				// exit
				break
			}

			if err != nil {

				p.errorLog.Printf("Incoming connection error %s", err.Error())

				// log error and continue
				continue
			}

			p.traceLog.Printf("New incoming connection %s", client.RemoteAddr().String())

			go p.handleConnection(client)
		}

		p.traceLog.Printf("Return routine")
		p.completeChan <- true
		// listener released
		p.state = 3
	}()

	p.traceLog.Printf("Return Run")
	return nil
}

func (p *mysqlProxy) Stop() error {

	if p.state == 3 {
		// already stopped
		return nil
	}

	close(p.stopChan)

	if p.state == 2 {

		// open connection to itself to unblock listener
		conn, err := net.Dial("tcp", p.proxyHost)

		if err != nil {
			return err
		}
		defer conn.Close()

		conn.Write([]byte{0, 0, 0})
	}

	return nil
}

func (p *mysqlProxy) IsStopped() bool {
	return p.state == 3
}
func (p *mysqlProxy) WaitStop() {
	// wait while Run routine adds value to this channel
	<-p.completeChan
}

// handleConnection ...
func (p *mysqlProxy) handleConnection(client net.Conn) {
	defer p.traceLog.Printf("Close incoming connection from %s", client.RemoteAddr().String())

	defer client.Close()

	// New connection to MySQL is made per each incoming TCP request to proxy server.
	var server net.Conn
	var err error

	if strings.HasPrefix(p.mysqlHost, "/") {
		// unix socket connection
		server, err = net.Dial("unix", p.mysqlHost)
	} else {
		// tcp connection
		server, err = net.Dial("tcp", p.mysqlHost)
	}

	if err != nil {

		p.errorLog.Printf("Can not connect to mysql %s : Error: %s", p.mysqlHost, err.Error())

		// return error to client
		return
	}
	defer server.Close()

	p.traceLog.Printf("Connected to MySQL")

	sessionID := randString(10)

	requestFilter := p.getRequestManager(server, client, sessionID)

	// read request in parallel routine
	go io.Copy(requestFilter, client)
	// response manager will be connected to request manager
	responseFilter := p.getResponseManager(client, sessionID, requestFilter)

	// read response. Response will be first operation
	io.Copy(responseFilter, server)

}

// Build requestPacketParser object
func (p *mysqlProxy) getRequestManager(server net.Conn, client net.Conn, sessID string) *requestPacketParser {
	r := requestPacketParser{}
	r.server = server
	r.client = client
	r.sessionID = sessID
	r.protocol = protocolInfo{}
	r.requestCallback = p.requestCallback
	r.queryFilter = p.queryFilter
	r.traceLog = p.traceLog
	r.errorLog = p.errorLog
	return &r
}

// Build responsePacketParser object
func (p *mysqlProxy) getResponseManager(client net.Conn, sessID string, rParser *requestPacketParser) *responsePacketParser {
	r := responsePacketParser{}
	r.client = client
	r.sessionID = sessID
	r.responseCallback = p.responseCallback
	r.queryFilter = p.queryFilter
	r.traceLog = p.traceLog
	r.errorLog = p.errorLog
	r.initialResponseSet = false
	r.requestParser = rParser
	return &r
}

// Parse data sent from client to server
type requestPacketParser struct {
	server             net.Conn
	client             net.Conn
	sessionID          string
	initialResponseSet bool
	protocol           protocolInfo
	requestCallback    RequestQueryFilterCallback
	queryFilter        DBProxyFilter
	traceLog           *log.Logger
	errorLog           *log.Logger
}

// data posted from client to server
func (pp *requestPacketParser) Write(p []byte) (n int, err error) {
	n = len(p) // we will return this number for any action

	pp.traceLog.Printf("Request: %d bytes with type %x", len(p), getPacketType(p))

	if !pp.initialResponseSet {
		// clients sends data back on server's handshake

		pp.parseHandshakeClientResponse(p)
		pp.initialResponseSet = true
	}
	// pass request to server or return error response

	var clientErr error
	var customResponse CustomRequestActionInterface

	switch getPacketType(p) {

	case comStmtPrepare:
	case comQuery:

		decoded, err := decodeQueryRequest(p)

		if err == nil {
			// pass through filters
			if pp.queryFilter != nil {
				customResponse, clientErr = pp.queryFilter.RequestCallback(decoded.Query, pp.sessionID)
			}
			if customResponse == nil && clientErr == nil && pp.requestCallback != nil {
				customResponse, clientErr = pp.requestCallback(decoded.Query, pp.sessionID)
			}

			pp.traceLog.Printf("Request: %s", decoded)
		}
	}

	if clientErr != nil {
		// send error response to client
		pp.traceLog.Printf("Custom error response: %s", clientErr)
		customResponse = NewCustomErrorResponse(err.Error(), 3001)

	}
	if customResponse != nil {
		pp.traceLog.Printf("Custom response")

		if crp, ok := customResponse.(customRequestActionNeedProtocolInterface); ok {
			crp.setProtocolInfo(pp.protocol)
		}

		if customRequest, ok := customResponse.(customRequestModifyInterface); ok {

			// means return data back to client
			customRequest.setOriginalRequest(p)

			// request was modified. send it to a server
			packet := customResponse.getPacket()

			pp.traceLog.Printf("Send custom request to server. %d bytes", len(packet))

			io.Copy(pp.server, bytes.NewReader(packet))
		} else {

			packet := customResponse.getPacket()

			pp.traceLog.Printf("Send custom response to client. %d bytes", len(packet))

			io.Copy(pp.client, bytes.NewReader(packet))
		}

		return
	}
	// Default Action
	io.Copy(pp.server, bytes.NewReader(p))

	return
}

// extract protocol info from initial handshake
// this info will be needed to make custom responses later
func (pp *requestPacketParser) parseHandshake(packet []byte) {
	serverHandshake, err := decodeHandshakeV10(packet)

	if err != nil {
		pp.traceLog.Printf("Handshake error %s", err.Error())
		return
	}
	pp.protocol.serverInfo = serverHandshake
}

// extract client capabilities from a response on handshake from server
func (pp *requestPacketParser) parseHandshakeClientResponse(packet []byte) {
	clientHandshake, err := decodeHandshakeResponse41(packet)

	if err != nil {
		pp.traceLog.Printf("Client Handshake error %s", err.Error())
		return
	}
	pp.protocol.clientInfo = clientHandshake
}

// Response manager. Reads responses from server and sends to a client.
// This can be used to modify responses later. But now we don't modify it
// here we only can analise responses
type responsePacketParser struct {
	client             net.Conn
	sessionID          string
	initialResponseSet bool
	responseCallback   ResponseFilterCallback
	queryFilter        DBProxyFilter
	traceLog           *log.Logger
	errorLog           *log.Logger
	requestParser      *requestPacketParser
}

// Data received from server and it is time to send it to client
func (pp *responsePacketParser) Write(p []byte) (n int, err error) {
	pp.traceLog.Printf("Write to Response , bytes received %d, type %x\n", len(p), getPacketType(p))

	switch getPacketType(p) {

	case responseErr:
		decoded, _ := decodeErrResponse(p)
		pp.traceLog.Printf("Server response error %s", decoded)

		if pp.queryFilter != nil {
			pp.queryFilter.ResponseCallback(pp.sessionID, errors.New(decoded))
		}
		if pp.responseCallback != nil {
			pp.responseCallback(pp.sessionID, errors.New(decoded))
		}

	default:
		pp.traceLog.Printf("Response OK")
		if pp.queryFilter != nil {
			pp.queryFilter.ResponseCallback(pp.sessionID, nil)
		}
		if pp.responseCallback != nil {
			pp.responseCallback(pp.sessionID, nil)
		}

		if !pp.initialResponseSet {
			pp.traceLog.Printf("Initial handshake")
			pp.requestParser.parseHandshake(p)
			pp.initialResponseSet = true
		}
	}
	pp.traceLog.Printf("Send response to client. %d bytes", len(p))

	io.Copy(pp.client, bytes.NewReader(p))

	return len(p), nil
}
