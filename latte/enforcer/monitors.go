package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
)

func (enforcer *Enforcer) DispatchFork(fStr string) {

	logrus.Debugf("Fork string: [[%s]]", fStr)
	buf := bytes.NewBufferString(fStr)
	decoder := json.NewDecoder(buf)
	var e ForkEv
	if err := decoder.Decode(&e); err != nil {
		logrus.Error("error parsing the fork string, ", fStr)
		return
	}
	enforcer.fork <- e
}

func (enforcer *Enforcer) DispatchExec(eStr string) {
	logrus.Debugf("Exec string: [[%s]]", eStr)
	buf := bytes.NewBufferString(eStr)
	decoder := json.NewDecoder(buf)
	var e ExecEv
	if err := decoder.Decode(&e); err != nil {
		logrus.Error("error parsing the exec string, ", eStr)
		return
	}
	enforcer.exec <- e
}

func (enforcer *Enforcer) Dispatch(evStr string) {

	if strings.HasPrefix(evStr, ForkPrefix) {
		enforcer.DispatchFork(evStr[len(ForkPrefix):])
	} else if strings.HasPrefix(evStr, ExecPrefix) {
		enforcer.DispatchExec(evStr[len(ExecPrefix):])
	} else {
		logrus.Error("Unknown cmd line: [", evStr, "], skip")
	}

}

func (enforcer *Enforcer) Monitor(r io.ReadCloser) {
	reader := bufio.NewReader(r)
	buffer := bytes.Buffer{}
	for {
		data, prefix, err := reader.ReadLine()
		if err != nil {
			if data != nil && len(data) > 0 {
				buffer.Write(data)
			}
			if buffer.Len() > 0 {
				enforcer.Dispatch(buffer.String())
			}
			if err != io.EOF {
				logrus.Error("something wrong receiving output", err)
				enforcer.err <- err
			}
			break
		}
		buffer.Write(data)
		if !prefix {
			enforcer.logout.Write(buffer.Bytes())
			enforcer.logout.Write([]byte{'\n'})
			enforcer.Dispatch(buffer.String())
			buffer.Reset()
		}
	}
	enforcer.done <- true
}
