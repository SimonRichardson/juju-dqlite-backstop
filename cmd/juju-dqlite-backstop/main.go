// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/juju/collections/set"
	"github.com/juju/names/v4"

	"github.com/SimonRichardson/juju-dqlite-backstop/agent"
	"github.com/SimonRichardson/juju-dqlite-backstop/database"
	"github.com/SimonRichardson/juju-dqlite-backstop/database/dqlite"
	"github.com/SimonRichardson/juju-dqlite-backstop/version"
)

var controllerPrompt = `
This program should only be used to recover from specific Dqlite
HA related problems. Casual use is strongly discouraged.
Irreversible damage may be caused to a Juju deployment through 
improper use of this tool.

Aside from limited cases, this program should not be run while Juju
controller machine agents are running.

Ok to proceed?`[1:]

type commandLineArgs struct {
	controllerTag   string
	agentConfigPath string
	doPrompt        bool
}

func main() {
	checkErr("setupLogging", setupLogging())
	args := commandLine()

	if args.doPrompt && !promptYN(controllerPrompt) {
		return
	}

	t, err := names.ParseTag(args.controllerTag)
	checkErr("parse controller tag", err)

	agent, err := agent.ReadConfig(agent.ConfigPath(args.agentConfigPath, t))
	checkErr("read agent config", err)

	nodeManager := database.NewNodeManager(agent, logger)
	_, err = nodeManager.EnsureDataDir()
	checkErr("ensure data dir", err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodeInfo, err := nodeManager.ClusterServers(ctx)
	checkErr("get cluster servers", err)

	// If we're the only node in the cluster, then we're done.
	if len(nodeInfo) == 1 {
		fmt.Fprintf(os.Stderr, "unable to perform dqlite backstop with one node\n")
		os.Exit(0)
	}

	addresses, err := agent.APIAddresses()
	checkErr("get api addresses", err)

	// If the number of addresses matches the number of nodes, then we're done.
	if numAddresses := len(addresses); numAddresses == len(nodeInfo) {
		fmt.Fprintf(os.Stderr, "unable to perform dqlite backstop action where the number addresses match the number of nodes\n")
		os.Exit(0)
	} else if numAddresses > 1 {
		fmt.Fprintf(os.Stderr, "unable to perform dqlite backstop action where the number addresses is greater than one\n")
		os.Exit(0)
	}

	hosts := set.NewStrings()
	for _, info := range nodeInfo {
		host, _, err := net.SplitHostPort(info.Address)
		checkErr("split api host port", err)
		hosts.Add(host)
	}

	var (
		leader dqlite.NodeInfo
		found  bool
	)
	for _, info := range nodeInfo {
		host, _, err := net.SplitHostPort(info.Address)
		checkErr("split node host port", err)
		if hosts.Contains(host) {
			leader = info
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "unable to find leader node\n")
		os.Exit(1)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = nodeManager.SetClusterServers(ctx, []dqlite.NodeInfo{leader})
	checkErr("set cluster servers", err)

	fmt.Println("dqlite backstop action complete")
	fmt.Println("please restart the controller machine agents using:")
	fmt.Println("")
	fmt.Printf("\tsystemctl restart jujud-%s.service\n", args.controllerTag)
	fmt.Println("")
}

func checkErr(label string, err error) {
	if err != nil {
		logger.Errorf("%s: %s", label, err)
		os.Exit(1)
	}
}

func commandLine() commandLineArgs {
	flags := flag.NewFlagSet("dqlite-backstop", flag.ExitOnError)
	var a commandLineArgs
	yes := flags.Bool("yes", false, "answer 'yes' to prompts")
	showVersion := flags.Bool("version", false, "show version")
	path := flags.String("path", agent.DefaultPaths.DataDir, "path to agent config")

	flags.Parse(os.Args[1:])

	if *showVersion {
		fmt.Fprintf(os.Stderr, "%s\n%s-%s\n", version.Version, version.GitCommit, version.GitTreeState)
		os.Exit(0)
	}

	args := flags.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <tag>\n", os.Args[0])
		os.Exit(1)
	}

	a.doPrompt = !*yes
	a.controllerTag = args[0]
	a.agentConfigPath = *path

	return a
}

func promptYN(question string) bool {
	fmt.Printf("%s [y/n] ", question)
	os.Stdout.Sync()
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	switch strings.ToLower(scanner.Text()) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
