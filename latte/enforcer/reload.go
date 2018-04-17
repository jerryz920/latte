package main

import (
	"fmt"

	"github.com/jerryz920/utils/random/conferences/eurosys18"
	"github.com/sirupsen/logrus"
)

// failure in reload will just stop the executable
func (enforcer *Enforcer) Reload() {
	eurosys18.RestartStore(false)
	pstore, err := eurosys18.NewStore(PrincipalStoreName, false)
	if err != nil {
		logrus.Fatal("error creating principal store! ", err)
	}
	enforcer.principalStore = pstore

	istore, err := eurosys18.NewStore(ImageStoreName, true)
	if err != nil {
		logrus.Fatal("error creating image store! ", err)
	}
	enforcer.imageStore = istore

	plstore, err := eurosys18.NewStore(PolicyStoreName, false)
	if err != nil {
		logrus.Fatal("error creating policy store! ", err)
	}
	enforcer.policyStore = plstore

	for _, k := range enforcer.policyStore.Keys() {
		enforcer.LoadPolicy(k)
	}
	for _, k := range enforcer.principalStore.Keys() {
		var pid int
		if n, err := fmt.Sscan(k, &pid); n != 1 || err != nil {
			logrus.Error("fail to load pid from key %s: %s", k, err)
			continue
		}
		enforcer.LoadProc(pid)
	}
	for _, k := range enforcer.imageStore.Keys() {
		/// local cached image information
		enforcer.LoadImage(k)
	}
}
