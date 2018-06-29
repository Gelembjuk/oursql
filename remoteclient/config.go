package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gelembjuk/oursql/lib/remoteclient"
)

func GetAppInput() (remoteclient.AppInput, error) {
	input := remoteclient.AppInput{}

	if len(os.Args) < 2 {
		return input, nil
	}

	input.Command = os.Args[1]

	cmd := flag.NewFlagSet(input.Command, flag.ExitOnError)

	cmd.StringVar(&input.Address, "address", "", "Address of operation")
	cmd.StringVar(&input.Address, "from", "", "Address to send money from")
	cmd.StringVar(&input.ToAddress, "to", "", "Address to send money to")
	cmd.IntVar(&input.NodePort, "nodeport", 0, "Node Server port")
	cmd.StringVar(&input.NodeHost, "nodehost", "", "Node Server Host")
	cmd.Float64Var(&input.Amount, "amount", 0, "Amount money to send")
	cmd.StringVar(&input.LogDest, "logdest", "file", "Destination of logs. file or stdout")

	datadirPtr := cmd.String("configdir", "", "Location of data files, config")

	err := cmd.Parse(os.Args[2:])

	if err != nil {
		log.Panic(err)
	}

	if *datadirPtr != "" {
		input.ConfigDir = *datadirPtr
		if input.ConfigDir[len(input.ConfigDir)-1:] != "/" {
			input.ConfigDir += "/"
		}
	}
	if input.ConfigDir == "" {
		input.ConfigDir = "config/"
	}

	if _, err := os.Stat(input.ConfigDir); os.IsNotExist(err) {
		os.Mkdir(input.ConfigDir, 0755)
	}

	// read config file . command line arguments are more important than a config

	file, errf := os.Open(input.ConfigDir + "config.json")

	if errf != nil && !os.IsNotExist(errf) {
		// error is bad only if file exists but we can not open to read
		return input, errf
	}
	if errf == nil {
		config := remoteclient.AppInput{}
		// we open a file only if it exists. in other case options can be set with command line
		decoder := json.NewDecoder(file)
		err := decoder.Decode(&config)

		if err != nil {
			return input, err
		}

		if input.NodeHost == "" && config.NodeHost != "" {
			input.NodeHost = config.NodeHost
		}
		if input.NodePort == 0 && config.NodePort > 0 {
			input.NodePort = config.NodePort
		}
		if input.Address == "" && config.Address != "" {
			input.Address = config.Address
		}
	}

	return input, nil
}

func checkNeedsHelp(c remoteclient.AppInput) bool {
	if c.Command == "help" || c.Command == "" {
		return true
	}
	return false
}
func checkConfigUpdateNeeded(c remoteclient.AppInput) bool {
	if c.Command == "setnode" {
		return true
	}
	return false
}

func updateConfig(datadir string, c remoteclient.AppInput) error {
	config := remoteclient.AppInput{}

	configfile := datadir + "config.json"

	file, errf := os.Open(configfile)

	if errf != nil && !os.IsNotExist(errf) {
		// error is bad only if file exists but we can not open to read
		return errf
	}
	if errf == nil {
		// we open a file only if it exists. in other case options can be set with command line
		decoder := json.NewDecoder(file)
		err := decoder.Decode(config)

		file.Close()

		if err != nil {
			return err
		}
	}

	if c.Command == "setnode" {
		config.NodeHost = c.NodeHost
		config.NodePort = c.NodePort
	}

	// convert back to JSON and save to config file
	file, errf = os.OpenFile(configfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if errf != nil {
		return errf
	}

	encoder := json.NewEncoder(file)

	err := encoder.Encode(&config)

	if err != nil {
		return err
	}

	file.Close()
	return nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  help - Prints this help")
	fmt.Println("  == Any of next commands can have optional argument [-configdir /path/to/dir] [-logdest stdout] ==")
	fmt.Println("  createwallet\n\t- Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  showunspent -address ADDRESS\n\t- Displays the list of all unspent transactions and total balance")
	fmt.Println("  showhistory -address ADDRESS\n\t- Displays the wallet history. All In?Out transactions")
	fmt.Println("  getbalance -address ADDRESS\n\t- Get balance of ADDRESS")
	fmt.Println("  listaddresses\n\t- Lists all addresses from the wallet file")
	fmt.Println("  listbalances\n\t- Lists all addresses from the wallet file and show balance for each")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT\n\t- Send AMOUNT of coins from FROM address to TO. ")
	fmt.Println("  setnode -nodehost HOST -nodeport PORT\n\t- Saves a node host and port to configfile. ")
}
