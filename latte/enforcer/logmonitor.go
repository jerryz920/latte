package main

import "io"

func (enforcer *Enforcer) ErrMonitor(r io.ReadCloser) {
	io.Copy(enforcer.logerr, r)
}
