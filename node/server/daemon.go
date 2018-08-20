package server

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gelembjuk/oursql/lib/net"
	"github.com/gelembjuk/oursql/lib/remoteclient"
	"github.com/gelembjuk/oursql/lib/utils"
	"github.com/gelembjuk/oursql/node/config"
	"github.com/gelembjuk/oursql/node/nodemanager"
)

type NodeDaemon struct {
	Port        int
	Host        string
	ConfigDir   string
	Server      *NodeServer
	Logger      *utils.LoggerMan
	Node        *nodemanager.Node
	DBProxyAddr string
	DBAddr      string
}

func (n *NodeDaemon) Init() error {
	n.createServer()

	return nil
}

// Build a pid file path

func (n *NodeDaemon) getServerPidFile() string {
	return n.ConfigDir + config.PidFileName
}

// Create node server object. It will set tcp port and handle requests from other nodes

func (n *NodeDaemon) createServer() error {
	if n.Server != nil {
		return nil
	}

	server := NodeServer{}

	server.NodeAddress.Port = n.Port
	server.NodeAddress.Host = n.Host

	server.ConfigDir = n.ConfigDir

	server.StopMainChan = make(chan struct{})
	server.StopMainConfirmChan = make(chan struct{})

	server.Logger = n.Logger

	server.Transit.Init(n.Logger)

	server.Node = n.Node

	server.DBProxyAddr = n.DBProxyAddr
	server.DBAddr = n.DBAddr

	n.Server = &server

	return nil
}

// Check state of pid file. To know if a node server is running or not

func (n *NodeDaemon) checkPIDFile() error {
	// check if daemon already running.
	if _, err := os.Stat(n.getServerPidFile()); err == nil {
		n.Logger.Error.Printf("Already running or %s file exist.", n.getServerPidFile())

		isfine := true
		// check if process is really running
		ProcessID, _, _, _, err := n.loadPIDFile()

		if err == nil && ProcessID > 0 {

			if !n.checkProcessExists(ProcessID) {
				// process is not found
				// remove PID file
				isfine = false
			}
		} else {
			// pid file has wrong format
			isfine = false
		}

		if isfine {
			return errors.New("Already running or PID file exists")
		} else {
			os.Remove(n.getServerPidFile())
		}

	}
	return nil
}

// Help function to ferify input arguments on start

func (n *NodeDaemon) checkArgumentsAreFine() error {

	if n.Port < 1 {
		return errors.New("Node port is not provided")
	}

	if n.Port < 1025 || n.Port > 65536 {
		return errors.New("Node port has wrong value. Must be more 1024 and less 65536")
	}

	if n.Server.Node.MinterAddress == "" {
		return errors.New("Minter Address is not provided")
	}
	// check if wallet is good and exists in wallets
	winput := remoteclient.AppInput{}
	winput.ConfigDir = n.ConfigDir

	walletscli := remoteclient.WalletCLI{}

	walletscli.Init(n.Logger, winput)

	_, err := walletscli.WalletsObj.GetWallet(n.Server.Node.MinterAddress)

	if err != nil {
		return errors.New("Minter Address can not be loaded from wallet. Does it exist?")
	}

	return nil
}

// Starts a node in daemon mode. Creates new process and this process exists

func (n *NodeDaemon) StartServer() error {
	// check if daemon already running.
	err := n.checkPIDFile()

	if err != nil {
		return err
	}

	err = n.checkArgumentsAreFine()

	if err != nil {
		return err
	}
	logsstate := n.Logger.GetState()

	command := os.Args[0] + " " + config.Daemonprocesscommandline + " " +
		"-configdir=" + n.ConfigDir + " " +
		"-minter=" + n.Server.Node.MinterAddress + " " +
		"-port=" + strconv.Itoa(n.Port) + " " +
		"-host=" + n.Host + " " +
		"-logs=" + logsstate

	n.Logger.Trace.Println("Execute command : ", command)

	n.Logger.DisableLogging()

	cmd := exec.Command(os.Args[0], config.Daemonprocesscommandline,
		"-configdir="+n.ConfigDir,
		"-minter="+n.Server.Node.MinterAddress,
		"-port="+strconv.Itoa(n.Port),
		"-host="+n.Host,
		"-logs="+logsstate)
	cmd.Start()
	n.Logger.Trace.Println("Daemon process ID is : ", cmd.Process.Pid)
	n.savePIDFile(cmd.Process.Pid, n.Port, "", "n")

	i := 0

	for {
		time.Sleep(1 * time.Second)

		_, _, _, startres, err := n.loadPIDFile()

		if err != nil {
			break
		}
		if startres != "y" && startres != "n" {
			n.Logger.Trace.Println("STart error: " + startres)
			return errors.New(startres)
		}

		if startres == "y" || i > 10 {
			break
		}
		i++
	}

	return nil
}

// Runs a node server without a daemon. This can help to debug a node

func (n *NodeDaemon) StartServerInteractive() error {
	// check if daemon already running.
	err := n.checkPIDFile()

	if err != nil {
		return err
	}

	err = n.checkArgumentsAreFine()

	if err != nil {
		return err
	}

	pid := os.Getpid()

	n.Logger.Trace.Println("Process ID is : ", pid)

	authstr, err := n.savePIDFile(pid, n.Port, "", "y")

	if err != nil {
		return err
	}

	n.Server.NodeAuthStr = authstr

	err = n.DaemonizeServer()

	os.Remove(n.getServerPidFile())

	if err != nil {
		return err
	}

	return nil
}

// Stops a node daemon. Finds a process and kills it.

func (n *NodeDaemon) StopServer() error {
	ProcessID, _, _, _, err := n.loadPIDFile()

	if err == nil && ProcessID > 0 {

		process, err := os.FindProcess(ProcessID)

		if err != nil {
			n.Logger.Error.Printf("Unable to find process ID [%v] with error %v \n", ProcessID, err)
			return nil
		}

		// remove PID file
		os.Remove(n.getServerPidFile())

		n.Logger.Trace.Printf("Killing process ID [%v] now.\n", ProcessID)
		// kill process and exit immediately
		//err = process.Kill()
		err = process.Signal(syscall.SIGTERM)

		if err != nil {
			n.Logger.Error.Printf("Unable to kill process ID [%v] with error %v \n", ProcessID, err)
			return nil
		}

		n.Logger.Trace.Printf("Killed process ID [%v]\n", ProcessID)

		return nil

	} else if err != nil {
		return err
	}
	n.Logger.Warning.Println("Not running.")
	return nil
}

// Makes a daemon process. It starts node server and waits for interrupt signal when to exit.
func (n *NodeDaemon) DaemonizeServer() error {
	n.Logger.Trace.Println("Daemon process runs")

	_, _, authstr, _, _ := n.loadPIDFile()

	n.Server.NodeAuthStr = authstr

	// the channel to notify main thread about all work done on kill signal
	theendchan := make(chan struct{})

	// Make arrangement to remove PID file upon receiving the SIGTERM from kill command
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func(server *NodeServer) {
		signalType := <-ch
		signal.Stop(ch)

		// before terminating.
		n.Logger.Trace.Println("Received signal type : ", signalType)

		close(server.StopMainChan)

		// to force server to try to handle next command if there were no input connects
		// if we don't do this it will stay in "Accepting" mode and can not real channel
		n.Logger.Trace.Println("Send void command on port ", server.NodeAddress.Port)
		serverAddr := net.NodeAddr{"localhost", server.NodeAddress.Port}

		nodeclient := server.GetClient()

		err := nodeclient.SendVoid(serverAddr)

		if err != nil {
			server.Logger.Error.Println(err.Error())
		}

		n.Logger.Trace.Println("Waiting confirmation from main routine")
		<-server.StopMainConfirmChan

		// this is time to complete everything, flush to disk etc

		// remove PID file
		os.Remove(n.getServerPidFile())

		n.Logger.Trace.Println("Daemon routine complete")

		close(theendchan)

		return

	}(n.Server)
	n.Logger.Trace.Println("Starting server")

	serverStartResult := make(chan string)

	// this function wil wait to confirm server started
	go n.waitServerStarted(serverStartResult)

	err := n.Server.StartServer(serverStartResult)

	if err == nil {
		<-theendchan
	} else {
		// if server returned error it means it was not correct closing.
		// so ending channel was not filled
		n.Logger.Trace.Println("Server stopped with error: " + err.Error())
	}

	// white while response from server si read in "wait" function
	<-serverStartResult

	n.Logger.Trace.Println("Node Server Stopped")

	return nil
}

// check while sever started
func (n *NodeDaemon) waitServerStarted(serverStartResult chan string) {

	result := <-serverStartResult

	if result == "" {
		result = "y"
	}
	pid, port, authstring, _, err := n.loadPIDFile()

	if err == nil {

		// save status to know when server started
		n.savePIDFile(pid, port, authstring, result)
	}

	close(serverStartResult)
}

// Save PID file for a process

func (n *NodeDaemon) savePIDFile(pid int, port int, authstr string, startresult string) (string, error) {

	file, err := os.Create(n.getServerPidFile())

	if err != nil {
		n.Logger.Error.Printf("Unable to create pid file : %v\n", err)
		return "", err
	}

	defer file.Close()

	// generate some random string. it will be used to auth local network requests
	if authstr == "" {
		authstr = utils.RandString(net.CommandLength) // we use same length as for network commands, but this is not related
	}

	if len(startresult) > 1 {
		startresult = base64.StdEncoding.EncodeToString([]byte(startresult))
	}

	_, err = file.WriteString(strconv.Itoa(pid) + " " + strconv.Itoa(port) + " " + authstr + " " + startresult)

	if err != nil {
		n.Logger.Error.Printf("Unable to create pid file : %v\n", err)
		return "", err
	}

	file.Sync() // flush to disk

	return authstr, nil
}

// Laads PID file.
func (n *NodeDaemon) loadPIDFile() (int, int, string, string, error) {

	if _, err := os.Stat(n.getServerPidFile()); err == nil {
		// get running port from pid file
		pidfilecontentsbytes, err := ioutil.ReadFile(n.getServerPidFile())

		if err != nil {
			return 0, 0, "", "", err
		}

		pidfilecontents := string(pidfilecontentsbytes)

		parts := strings.Split(pidfilecontents, " ") // port is after pid and space in this text

		if len(parts) == 4 {
			portstring := parts[1]
			pidstring := parts[0]
			authstring := parts[2]
			startresult := parts[3]

			if len(startresult) > 1 {
				sDec, _ := base64.StdEncoding.DecodeString(startresult)
				startresult = string(sDec)
			}

			port, err := strconv.Atoi(portstring)
			if err != nil {
				return 0, 0, "", "", err
			}

			pid, errp := strconv.Atoi(pidstring)

			if errp != nil {
				return 0, 0, "", "", errp
			}

			return pid, port, authstring, startresult, nil
		}
		return 0, 0, "", "", errors.New("PID file wrong format")
	}

	return -1, 0, "", "", nil
}

/*
* Returns state of a server. Detects if it is running
 */
func (n *NodeDaemon) GetServerState() (bool, int, int, error) {
	ProcessID, Port, _, _, err := n.loadPIDFile()

	if err == nil && ProcessID > 0 {

		if n.checkProcessExists(ProcessID) {

			return true, ProcessID, Port, nil // server is running
		}

		// remove pid file. no sense to keep it if process failed
		os.Remove(n.getServerPidFile())
	}

	return false, 0, 0, nil
}

func (n *NodeDaemon) checkProcessExists(pid int) bool {
	p, err := os.FindProcess(pid)

	if err != nil {
		return false
	}

	err = p.Signal(syscall.Signal(0))

	if err == nil {
		return true
	}
	return false

}

// Returns port and auth string of currently running process
func (n *NodeDaemon) GetRunningProcessInfo() (int, string) {
	_, port, authstr, _, err := n.loadPIDFile()

	if err == nil && port > 0 {
		return port, authstr
	}
	return 0, ""
}
