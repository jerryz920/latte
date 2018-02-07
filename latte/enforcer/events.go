package main

type ForkEv struct {
	Parent int
	Child  int
	Cmd    string `json:"cmd,omitempty"`
}

type ExecEv struct {
	Pid  int
	Cmd  string
	Exec string `json:"exec"`
}
