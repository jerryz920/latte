package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"

	jsys "github.com/jerryz920/utils/goutils/sys"
	"github.com/jerryz920/utils/random/conferences/eurosys18"
	"github.com/sirupsen/logrus"
)

// #define _GNU_SOURCE
// #include <unistd.h>
import "C"

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	flag.Parse()
	args := flag.Args()
	logrus.Info("args are: ", args)
	if len(args) < 4 {
		logrus.Fatal("usage: latte_exec pname policy_file image_hash original_exec args...")
	}

	conn, err := net.Dial("unix", "/var/run/latte/event.sock")
	if err != nil {
		logrus.Error("can not connect to the daemon")
		return
	}

	/// send the policy file over unix
	f, err := os.Open(args[1])
	if err != nil {
		logrus.Error("can not read policy file: ", err)
	}

	unixconn := conn.(*net.UnixConn)
	pname := args[0]
	if len(pname) > eurosys18.PrincipalNameLimit {
		pname = pname[:eurosys18.PrincipalNameLimit]
	}
	req := fmt.Sprintf("{\"principal\": \"%s\", \"image\": \"%s\"}", args[0],
		args[2])

	if err = jsys.SendFdMsg(unixconn, []byte{byte(len(req))}, int(f.Fd())); err != nil {
		logrus.Error("fail sending policy file length: ", err)
	} else {
		///wait for approval message
		if n, err := conn.Write([]byte(req)); err != nil || n != len(req) {
			logrus.Error("fail sending policy file: ", err)
		} else {

			data := make([]byte, 1)
			n, err := conn.Read(data)
			if err != nil || n != 1 {
				logrus.Error("something wrong when sending policy file, err info:", err)
				return
			}
			// something wrong in reading policy file, execution rejected
			if data[0] == 0 {
				logrus.Error("launching new principal rejected!")
				os.Exit(1)
			}
		}
	}

	f.Close()
	conn.Close()

	/// Drop privilege first
	ruid := os.Getuid()
	rgid := os.Getgid()
	syscall.Setresuid(ruid, ruid, ruid)
	syscall.Setresgid(rgid, rgid, rgid)

	logrus.Debug("Could execute in place now!", args[3:])
	err = syscall.Exec(args[3], args[3:], os.Environ())
	logrus.Error(err)
}
