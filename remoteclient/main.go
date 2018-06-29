package main

import (
	"fmt"
	"os"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/lib/remoteclient"
	"github.com/gelembjuk/oursql/lib/utils"
)

func main() {
	input, ierr := GetAppInput()

	if ierr != nil {
		fmt.Printf("Error: %s\n", ierr.Error())
		os.Exit(0)
	}
	fmt.Printf("%s Wallet - %s\n\n", lib.ApplicationTitle, lib.ApplicationVersion)
	if checkNeedsHelp(input) {
		printUsage()
		os.Exit(0)
	}
	if checkConfigUpdateNeeded(input) {
		err := updateConfig(input.ConfigDir, input)
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
		} else {
			fmt.Println("Config updated")
		}

		os.Exit(0)
	}

	logger := utils.CreateLogger()

	if input.LogDest != "stdout" {
		logger.LogToFiles(input.ConfigDir, "log_trace.txt", "log_info.txt", "log_warning.txt", "log_error.txt")
	}

	walletscli := remoteclient.WalletCLI{}
	walletscli.Init(logger, input)

	walletscli.NodeMode = false

	err := walletscli.ExecuteCommand()

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
