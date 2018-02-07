package main

import "github.com/sirupsen/logrus"

func (enforcer *Enforcer) handleExec(e ExecEv) {
	logrus.Infof("handle exec: %d %s", e.Pid, e.Exec)
	/// if the ID is not marked as skip, check the cmdline
	// skip if its hash equals to builder or execer
	// else, find the policy using its parent principal's information
	// find the image using sha1sum of this binary
	// mark the sha1sum for later usage...
	// then check if the whitelist matches the sha1sum
	hash := Gethash(e.Pid, e.Exec)
	if hash == "" {
		logrus.Errorf("EMISSING %s %d #exe gone after execution", e.Exec, e.Pid)
		return
	}
	if hash == ExecWrapperHash {
		logrus.Info("principal %d execution: %s", e.Pid, e.Cmd)
	} else if hash == BuildWrapperHash {
		logrus.Info("build task %d execution: %s", e.Pid, e.Cmd)
		if p, ok := enforcer.procs[e.Pid]; ok {
			p.IsBuild = true
		} else {
			logrus.Error("quirk: build invoked when monitor is not present. Fixing")
			enforcer.CreateNewProc(e.Pid, true, SystemPrincipal, SystemImage)
		}
	} else {
		/// normal execution
		var p *Proc
		var ok bool
		enforcer.procLock.Lock()
		p, ok = enforcer.procs[e.Pid]
		enforcer.procLock.Unlock()
		if ok {
			// principal not set
			if p.Principal == "" {
				logrus.Debugf("principal not set")
				p.Principal = SystemPrincipal
				p.Image = SystemImage
				p.IsBuild = false
				enforcer.StoreProc(e.Pid)
			}
			p.Cmd = e.Cmd

		} else {
			p = enforcer.CreateNewProc(e.Pid, false, SystemPrincipal, SystemImage)
		}
		//index, err := enforcer.LookupImageAndCache(hash)

		logrus.Infof("hash: %s, pid: %d, image: %s", hash, e.Pid, p.Image)

		if !enforcer.Eurosys18Enforce(hash, e.Pid, p, 0) {
			logrus.Errorf("EVIOLATE %s %d %s #content not allowed", hash, e.Pid, p.Image)
			/// async poison if occurs
		}

		//enforcer.policyLock.Lock()
		//if policy, ok := enforcer.policies[p.Principal]; ok {
		//	if !enforcer.PolicyEnforced(index.Property, p.IsBuild, policy) {
		//		logrus.Error("EVIOLATE %s %d %s %s #policy violated",
		//			e.Exec, e.Pid, hash, p.Principal)
		//	}
		//}
		//enforcer.policyLock.Unlock()
	}

}
