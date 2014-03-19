package main

import (
	longshoreman "./longshoreman"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var commands = []string{"repull", "restart", "deploy"}

func usageError(msg string) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", msg)
	usage()
}

func usage() {
	fmt.Fprint(os.Stderr, "Usage: longshoreman [OPTIONS] COMMAND\n\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	hostsArg := flag.String("hosts", "", "comma seperated list of IP:PORT pairs (REQUIRED)")
	imageArg := flag.String("image", "", "image name to deploy (REQUIRED)")
	commandArg := flag.String("command", "deploy", "command to run [repull, restart, deploy]")
	flag.Parse()

	if *imageArg == "" {
		usageError("Missing required -image argument")
	}

	if *hostsArg == "" {
		usageError("Missing required -hosts argument")
	}

	validCommand := false
	for _, c := range commands {
		if *commandArg == c {
			validCommand = true
			break
		}
	}
	if !validCommand {
		usageError("Invalid -command argument")
	}

	hosts := strings.Split(*hostsArg, ",")
	for _, host := range hosts {
		ip_port := strings.Split(host, ":")
		if len(ip_port) != 2 {
			usageError("Invalid IP:PORT pair provided")
		}
	}

	longshoreman := longshoreman.New(hosts, *imageArg)

	if *commandArg == "repull" {
		runRepull(longshoreman)
	} else if *commandArg == "restart" {
		runRestart(longshoreman)
	} else if *commandArg == "deploy" {
		runRepull(longshoreman)
		runRestart(longshoreman)
	}
}

func runRepull(longshoreman *longshoreman.Longshoreman) {
	errs := longshoreman.Repull()
	if len(errs) > 0 {
		log.Fatal(errs)
	}
}

func runRestart(longshoreman *longshoreman.Longshoreman) {
	errs := longshoreman.Restart()
	if len(errs) > 0 {
		log.Fatal(errs)
	}
}
