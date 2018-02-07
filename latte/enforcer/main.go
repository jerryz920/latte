package main

import (
	"flag"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	ExecHash  = "@exechash"
	BuildHash = "@buildhash"
)

func main() {
	flag.Parse()
	args := flag.Args()
	logrus.Info("Latte Code Integrity Enforcer")
	if len(args) < 2 {
		logrus.Fatal("usage: ./enforcer image_server_address image_hash")
	}
	pid := os.Getpid()
	exechash := Gethash(pid, ExecWrapperPath)
	if exechash == "" {
		logrus.Fatal("can not obtain exec hash")
	}
	buildhash := Gethash(pid, BuildWrapperPath)
	if exechash == "" {
		logrus.Fatal("can not obtain build hash")
	}
	ExecWrapperHash = exechash
	BuildWrapperHash = buildhash

	enforcer, err := NewEnforcer(args[0], args[1])
	if err != nil {
		logrus.Fatal("error initializing the enforcer: ", err)
	}
	enforcer.LaunchMonitors()
}
