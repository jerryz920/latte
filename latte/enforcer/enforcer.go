package main

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	eurosys18 "github.com/jerryz920/utils/random/conferences/eurosys18"
	"github.com/sirupsen/logrus"
)

type Enforcer struct {
	imageServerAddr string /// metadata service address
	err             chan error
	done            chan bool
	fork            chan ForkEv
	exec            chan ExecEv
	images          map[string]*Image
	procs           map[int]*Proc
	policies        map[string]*Policy
	logout          *os.File
	logerr          *os.File
	policyStore     eurosys18.Store
	principalStore  eurosys18.Store
	imageStore      eurosys18.Store
	imageLock       sync.Mutex
	policyLock      sync.Mutex
	procLock        sync.Mutex
	/// how many seconds to wait for the background downloading task
	backgroundWaitSec int
}

func (enforcer *Enforcer) LaunchMonitors() {
	sigs := make(chan os.Signal, 1)

	cmd := exec.Command("startmon", "--exec", "--fork")

	stream, err := cmd.StdoutPipe()
	if err != nil {
		logrus.Error("can not get pipe", err)
		return
	}

	errstream, err := cmd.StderrPipe()
	if err != nil {
		logrus.Error("can not get err pipe", err)
		return
	}

	if err := cmd.Start(); err != nil {
		logrus.Error("can not start command ", err)
		return
	}

	go enforcer.ErrMonitor(errstream)
	go enforcer.Monitor(stream)
	go enforcer.PolicyEventMonitor()
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logrus.Info("signal: ", sig)
		sigerr := cmd.Process.Signal(sig)
		if sigerr != nil {
			logrus.Error("signal error: ", sigerr)
		}
	}()

	defer enforcer.logout.Close()
	defer enforcer.logerr.Close()
	exit := false
	for !exit {
		select {
		case err := <-enforcer.err:
			{
				logrus.Error("error in monitor:", err)
				cmd.Process.Kill()
				exit = true
			}
		case fork := <-enforcer.fork:
			enforcer.handleFork(fork)
		case exec := <-enforcer.exec:
			enforcer.handleExec(exec)
		case <-enforcer.done:
			{
				logrus.Info("monitoring done")
				cmd.Process.Kill()
				exit = true
			}
		}
	}
	logrus.Info("monitor quits!")

}

func NewEnforcer(serverAddr string, systemImage string) (*Enforcer, error) {
	enforcer := &Enforcer{
		err:               make(chan error),
		exec:              make(chan ExecEv),
		fork:              make(chan ForkEv),
		done:              make(chan bool),
		images:            make(map[string]*Image),
		procs:             make(map[int]*Proc),
		policies:          make(map[string]*Policy),
		imageServerAddr:   serverAddr,
		backgroundWaitSec: BackgroundTaskTimeout,
	}

	if err := os.MkdirAll(EnforcerHome, 0755); err != nil {
		logrus.Fatal("can not create enforcer run directory, ", err)
	}

	fout, err := os.OpenFile(LogoutPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logrus.Error("open std log file: ", err)
		return nil, err
	}
	ferr, err := os.OpenFile(LogerrPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logrus.Error("open std err file: ", err)
		return nil, err
	}
	enforcer.logerr = ferr
	enforcer.logout = fout
	/// Trying to recover existed policies.
	enforcer.Reload()

	// config update
	SystemImage = systemImage

	return enforcer, nil
}
