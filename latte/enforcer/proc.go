package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type Proc struct {
	Cmd       string
	Hash      string
	Image     string
	Principal string
	IsBuild   bool
}

func (enforcer *Enforcer) StoreProc(p int) {
	enforcer.procLock.Lock()
	proc, ok := enforcer.procs[p]
	enforcer.procLock.Unlock()
	if ok {
		var isBuild string
		if proc.IsBuild {
			isBuild = "1"
		} else {
			isBuild = "0"
		}
		enforcer.principalStore.PutValues(fmt.Sprintf("%d", p),
			[]string{proc.Cmd, proc.Hash, proc.Image, proc.Principal, isBuild})
	}
}

func (enforcer *Enforcer) LoadProc(p int) {
	/// This will only be called at start up time
	pfields := enforcer.principalStore.GetValues(fmt.Sprintf("%d", p))
	proc := &Proc{
		Cmd:       pfields[0],
		Hash:      pfields[1],
		Image:     pfields[2],
		Principal: pfields[3],
	}
	if pfields[4] == "1" {
		proc.IsBuild = true
	} else {
		proc.IsBuild = false
	}

	fpath, hash := Getexec(p)
	// Still the same command we assume!
	if fpath == proc.Image && hash == proc.Hash {
		enforcer.procs[p] = proc
	} else {
		logrus.Infof("process %d gone, before: %s, now %s", p, proc.Cmd, fpath)
	}
}

func (enforcer *Enforcer) GetProc(p int) *Proc {
	enforcer.procLock.Lock()
	defer enforcer.procLock.Unlock()
	if p, ok := enforcer.procs[p]; ok {
		return p
	}
	return nil
}

func (enforcer *Enforcer) CreateNewProc(pid int, isBuild bool, pname string, image string) *Proc {
	/// We don't care about exec/image/hash here. Fork is always allowed
	child := &Proc{
		IsBuild:   isBuild,
		Principal: pname,
		Image:     image,
	}
	enforcer.procLock.Lock()
	enforcer.procs[pid] = child
	enforcer.procLock.Unlock()
	enforcer.StoreProc(pid)
	return child
}
