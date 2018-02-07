package main

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
)

type PolicyRequest struct {
	Principal string
	Image     string
}

func (enforcer *Enforcer) handlePolicyConn(conn net.Conn) {
	unixconn := conn.(*net.UnixConn)
	defer unixconn.Close()
	f, err := unixconn.File()
	if err != nil {
		logrus.Error("can not get connection fd: ", err)
		return
	}
	cred, err := syscall.GetsockoptUcred(int(f.Fd()), syscall.SOL_SOCKET,
		syscall.SO_PEERCRED)

	logrus.Infof("new policy event connection from process %d", cred.Pid)
	// validate the policy event can only be called from latte_exec and
	// build_exec binary (which is set-uid program, and will drop root
	// after they have posted the policy to use for all its children.

	fpath, hash := Getexec(int(cred.Pid))
	if fpath == "" || hash == "" {
		logrus.Error("invalid sender: must linger before execution continues")
		return
	}

	b := make([]byte, 1)
	oob := make([]byte, 32)
	n, noob, _, _, err := unixconn.ReadMsgUnix(b, oob)
	logrus.Infof("receveid %d bytes, %d oob, err %v", n, noob, err)

	if noob > 0 {
		scms, err := syscall.ParseSocketControlMessage(oob[:noob])
		if err != nil {
			logrus.Error("error parsing SCM: ", err)
			return
		}

		scm := scms[0]
		fds, err := syscall.ParseUnixRights(&scm)

		if err != nil {
			logrus.Error("error parsing unix rights", err)
			return
		}
		logrus.Info("received fds: ", fds)

		if len(fds) == 0 {
			logrus.Error("received no fd")
			return
		}

		fd := fds[0]
		file := os.NewFile(uintptr(fd), "testfile")
		defer file.Close()
		decoder := json.NewDecoder(file)
		var policy Policy
		if err := decoder.Decode(&policy); err != nil {
			logrus.Error("error parsing policy file", err)
			unixconn.Write([]byte{0})
			return
		}
		//logrus.Infof("link path: %s, read data: %s", link, string(fdata))

		datalen := int(b[0])
		tmplen := datalen
		buf := bytes.Buffer{}
		data := make([]byte, datalen)
		for datalen != 0 {
			nbyte, err := unixconn.Read(data)
			if err != nil {
				logrus.Error("error receiving data")
				unixconn.Write([]byte{0})
				break
			}
			buf.Write(data)
			datalen -= nbyte
		}

		if buf.Len() != tmplen {
			logrus.Error("error receiving data")
			unixconn.Write([]byte{0})
			return

		}
		logrus.Infof("data: %s", buf.String())

		var req PolicyRequest
		decoder = json.NewDecoder(&buf)
		if err = decoder.Decode(&req); err != nil {
			logrus.Error("error parsing data")
			unixconn.Write([]byte{0})
			return
		}

		enforcer.policyLock.Lock()
		enforcer.policies[req.Principal] = &policy
		enforcer.policyLock.Unlock()
		enforcer.PutPolicy(req.Principal)

		enforcer.procLock.Lock()
		proc, ok := enforcer.procs[int(cred.Pid)]
		enforcer.procLock.Unlock()
		if ok {
			proc.Principal = req.Principal
			proc.Image = req.Image
			enforcer.StoreProc(int(cred.Pid))
		} else {
			enforcer.CreateNewProc(int(cred.Pid), hash == BuildWrapperHash,
				req.Principal, req.Image)
		}

		unixconn.Write([]byte{1})
		logrus.Info("allow to proceed")

	} else {
		unixconn.Write([]byte{0})
		logrus.Info("no control message received")
		return
	}

}

func (enforcer *Enforcer) PolicyEventMonitor() {
	if info, err := os.Stat(EventSocketPath); err == nil {
		// existed
		if (info.Mode() & os.ModeSocket) != 0 {
			logrus.Debug("clear event socket")
			os.Remove(EventSocketPath)
		} else {
			logrus.Fatal("event policy file exist but not socket, aborting")
		}
	}

	l, err := net.Listen("unix", EventSocketPath)
	if err != nil {
		logrus.Fatal("error starting the policy event listener: ", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			logrus.Error("error receiving policy event connection: ", err)
			if conn != nil {
				conn.Close()
			}
			continue
		}
		enforcer.handlePolicyConn(conn)
	}
}
