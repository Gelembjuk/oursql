package dbproxy

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
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
	server, err := net.Dial("tcp", p.mysqlHost)
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

	responseFilter := p.getResponseManager(client, sessionID)

	// read response. Response will be first operation
	io.Copy(responseFilter, server)

}

// Build requestPacketParser object
func (p *mysqlProxy) getRequestManager(server net.Conn, client net.Conn, sessID string) *requestPacketParser {
	return &requestPacketParser{server, client, sessID, p.requestCallback, p.queryFilter, p.traceLog, p.errorLog}
}

// Build responsePacketParser object
func (p *mysqlProxy) getResponseManager(client net.Conn, sessID string) *responsePacketParser {
	return &responsePacketParser{client, sessID, p.responseCallback, p.queryFilter, p.traceLog, p.errorLog}
}

type requestPacketParser struct {
	server          net.Conn
	client          net.Conn
	sessionID       string
	requestCallback RequestQueryFilterCallback
	queryFilter     DBProxyFilter
	traceLog        *log.Logger
	errorLog        *log.Logger
}

func (pp *requestPacketParser) Write(p []byte) (n int, err error) {
	pp.traceLog.Printf("Request: %d bytes", len(p))

	// pass request to server or return error response

	var clientErr error

	switch getPacketType(p) {

	case comStmtPrepare:
	case comQuery:

		decoded, err := decodeQueryRequest(p)

		if err == nil {
			// pass through filters
			if pp.queryFilter != nil {
				clientErr = pp.queryFilter.RequestCallback(decoded.Query, pp.sessionID)
			}
			if clientErr == nil && pp.requestCallback != nil {
				clientErr = pp.requestCallback(decoded.Query, pp.sessionID)
			}

			pp.traceLog.Printf("Request: %s", decoded)
		}
	}
	if clientErr == nil {
		io.Copy(pp.server, bytes.NewReader(p))
	} else {
		// send error response to client
		pp.traceLog.Printf("Custom error response: %s", clientErr)

		var errResp []byte

		if rerr, ok := clientErr.(ResponseError); ok {
			errResp = rerr.getMySQLError()
		} else {
			rerr := NewMySQLError(clientErr.Error(), 3001)
			errResp = rerr.getMySQLError()
		}
		pp.traceLog.Printf("Send response to client on custom error. %d bytes", len(errResp))
		io.Copy(pp.client, bytes.NewReader(errResp))
	}

	return len(p), nil
}

type responsePacketParser struct {
	client           net.Conn
	sessionID        string
	responseCallback ResponseFilterCallback
	queryFilter      DBProxyFilter
	traceLog         *log.Logger
	errorLog         *log.Logger
}

func (pp *responsePacketParser) Write(p []byte) (n int, err error) {
	pp.traceLog.Printf("Write to Response , bytes received %d\n", len(p))

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

	}
	pp.traceLog.Printf("Send response to client. %d bytes", len(p))

	io.Copy(pp.client, bytes.NewReader(p))

	return len(p), nil
}
