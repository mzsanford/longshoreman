package main

import (
	longshoreman "./longshoreman"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

var commands = []string{"pull", "restart", "deploy", "stop", "list", "cat"}

const version = "0.1.1"

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
	commandArg := flag.String("command", "deploy", fmt.Sprintf("command to run. Valid commands: [%s]", strings.Join(commands, ", ")))
	pullTimeout := flag.Int("pull-timeout", 30, "seconds to wait for 'docker pull'")
	restartWaitTime := flag.Int("restart-time-limit", 10, "seconds to wait for container restart before sending kill")
	helpArg := flag.Bool("help", false, "display usage message")
	catFile := flag.String("file", "", "File to cat (only valid with the cat command)")
	verboseArg := flag.Bool("v", false, "Display debug logging")
	quietArg := flag.Bool("q", false, "Quiet mode. Only log warnings and errors")
	noColorArg := flag.Bool("no-color", false, "Do not colorize output if a terminal is detected")
	versionArg := flag.Bool("version", false, "Display version number and exit")
	flag.Parse()

	if *helpArg {
		usage()
	}

	if *versionArg {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	// TODO: Maybe this is ok for commandArg==list?
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

	if *commandArg == "cat" {
		if *catFile == "" {
			usageError("-file required with the cat command")
		}
	} else {
		if *catFile != "" {
			usageError("-file option only valid with the cat command")
		}
	}

	hosts := strings.Split(*hostsArg, ",")
	for _, host := range hosts {
		ip_port := strings.Split(host, ":")
		if len(ip_port) != 2 {
			usageError("Invalid IP:PORT pair provided")
		}
	}

	client := longshoreman.New(hosts, *imageArg)
	client.Config.PullTimeout = time.Duration(*pullTimeout) * time.Second
	client.Config.RestartTimeout = time.Duration(*restartWaitTime) * time.Second

	if *quietArg {
		client.Logger.LogLevel = longshoreman.LogLevelWarn
	} else if *verboseArg {
		client.Logger.LogLevel = longshoreman.LogLevelDebug
	}
	if *noColorArg {
		client.Logger.Colorize = false
	}

	if *commandArg == "pull" {
		runRepull(client)
	} else if *commandArg == "stop" {
		runStop(client)
	} else if *commandArg == "restart" {
		runRestart(client)
	} else if *commandArg == "deploy" {
		runRepull(client)
		runRestart(client)
	} else if *commandArg == "list" {
		runList(client)
	} else if *commandArg == "cat" {
		runCat(client, *catFile)
	}
}

func runRepull(client *longshoreman.Longshoreman) {
	errs := client.Pull()
	if len(errs) > 0 {
		fmt.Println(errs)
		os.Exit(1)
	}
}

func runRestart(client *longshoreman.Longshoreman) {
	errs := client.Restart()
	if len(errs) > 0 {
		fmt.Println(errs)
		os.Exit(1)
	}
}

func runStop(client *longshoreman.Longshoreman) {
	errs := client.Stop()
	if len(errs) > 0 {
		fmt.Println(errs)
		os.Exit(1)
	}
}

func runList(client *longshoreman.Longshoreman) {
	reports := make(chan longshoreman.HostStatus, 0)
	go func() {
		errs := client.List(reports)
		if len(errs) > 0 {
			fmt.Println(errs)
			os.Exit(1)
		}
	}()

	for hostStatus := range reports {
		for _, container := range hostStatus.Containers {
			status := "down"
			statusNote := ""
			if container.State.Running {
				status = "up"
				statusNote = fmt.Sprintf(" (%s)", HumanDuration(time.Since(container.State.StartedAt)))
			}
			fmt.Printf("%s/%s[%s]: %v%s\n", hostStatus.Host, ShortImage(container.Config.Image), container.Image[:15], status, statusNote)
		}
	}
}

func runCat(client *longshoreman.Longshoreman, path string) {
	reports := make(chan longshoreman.HostContents, len(client.Hosts))
	go func() {
		errs := client.Cat(path, reports)
		if len(errs) > 0 {
			fmt.Println(errs)
			os.Exit(1)
		}
	}()

	for hostContents := range reports {
		if len(client.Hosts) > 1 {
			fmt.Println("")
		}

		lines := strings.Split(strings.TrimSuffix(hostContents.Contents, "\n"), "\n")
		for _, line := range lines {
			fmt.Printf("%s: %s\n", hostContents.Host, line)
		}
	}
}

func ShortImage(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) == 3 {
		port_path := strings.SplitN(parts[1], "/", 2)
		return port_path[1] + ":" + parts[2]
	}
	return image
}

func HumanDuration(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds < 1 {
		return "just now"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Minutes()); minutes == 1 {
		return "about a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := int(d.Hours()); hours == 1 {
		remainder := minutes % 60
		return fmt.Sprintf("1 hour, %d minutes", remainder)
	} else if hours < 48 {
		remainder := minutes % 60
		return fmt.Sprintf("%d hours, %d minutes", hours, remainder)
	} else {
		remainder := hours % 24
		return fmt.Sprintf("%d days, %d hours", hours/24, remainder)
	}
}
