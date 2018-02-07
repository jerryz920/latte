package main

import "github.com/sirupsen/logrus"

func (enforcer *Enforcer) handleFork(e ForkEv) {
	logrus.Infof("handle fork: %d %d", e.Parent, e.Child)

	var isBuild bool
	var principal string
	var image string
	if p, ok := enforcer.procs[e.Parent]; ok {
		isBuild = p.IsBuild
		principal = p.Principal
		image = p.Image
	} else {
		// If we don't have its parent in the proc table, then its parent must
		// not be launched by anyone except for system principal: enforcer runs
		// right after system principal but before any other principals.
		// It is not possible that a process fork and dies quickly, because we
		// must have an event that its parent forks it. Otherwise its parent forks
		// it before enforcer starts, and it must be a system pricipal
		isBuild = false
		principal = SystemPrincipal
		image = SystemImage
	}
	enforcer.CreateNewProc(e.Child, isBuild, principal, image)
}
