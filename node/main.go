package main

import (
	"fmt"
	"os"

	"github.com/gelembjuk/oursql/lib"
	"github.com/gelembjuk/oursql/node/config"
)

func main() {
	// Parse input
	input, ierr := config.GetAppInput()

	if ierr != nil {
		// something went wrong when parsing input data
		fmt.Printf("Error: %s\n", ierr.Error())
		os.Exit(0)
	}

	if input.CheckNeedsHelp() {
		fmt.Printf("%s - %s\n\n", lib.ApplicationTitle, lib.ApplicationVersion)
		// if user requested a help, display it
		input.PrintUsage()
		os.Exit(0)
	}

	if input.CheckConfigUpdateNeeded() {
		fmt.Printf("%s - %s\n\n", lib.ApplicationTitle, lib.ApplicationVersion)
		// save config using input arguments
		input.UpdateConfig()
		os.Exit(0)
	}
	// create node client object
	// this will create all other objects needed to execute a command
	cli := getNodeCLI(input)

	if cli.isInteractiveMode() {
		fmt.Printf("%s - %s\n\n", lib.ApplicationTitle, lib.ApplicationVersion)
		// it is command to display results right now
		err := cli.ExecuteCommand()

		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		os.Exit(0)
	}

	if cli.isNodeManageMode() {
		// it is the command to manage node server
		err := cli.ExecuteManageCommand()

		if err != nil {
			fmt.Printf("Node Manage Error: %s\n", err.Error())
		}

		os.Exit(0)
	}

	fmt.Println("Unknown command!")
	input.PrintUsage()
}
