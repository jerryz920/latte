package main

import "path/filepath"

const (
	/// event from system monitor
	ForkPrefix = "Fork"
	ExecPrefix = "Exec"

	ImageStoreName     = "enforcer-image"
	PrincipalStoreName = "enforcer-principal"
	PolicyStoreName    = "enforcer-policy"
)

var (
	ExecWrapperPath       = "/usr/bin/lexec"
	BuildWrapperPath      = "/usr/bin/lbuild"
	EnforcerHome          = "/var/run/latte/"
	LogoutPath            = filepath.Join(EnforcerHome, "enforcer-stdout")
	LogerrPath            = filepath.Join(EnforcerHome, "enforcer-stderr")
	EventSocketPath       = filepath.Join(EnforcerHome, "event.sock")
	BackgroundTaskTimeout = 15
	SystemPrincipal       = "@system"
	SystemImage           = ""
	/// A bunch of configs, these two hash will be replaced
	// before actual build during the VM image construction process
	ExecWrapperHash  = "@exehash"
	BuildWrapperHash = "@buildhash"
)
