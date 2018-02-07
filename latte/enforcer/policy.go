package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

type Policy struct {
	Buildmode bool
	Whitelist []string
}

func (enforcer *Enforcer) PutPolicyBytes(name string, data []byte) {
	value := hex.EncodeToString(data)
	enforcer.policyStore.Put(name, value)
}

func (enforcer *Enforcer) PutPolicy(name string) error {
	enforcer.policyLock.Lock()
	p, ok := enforcer.policies[name]
	enforcer.policyLock.Unlock()
	if ok {
		buf := bytes.Buffer{}
		encoder := json.NewEncoder(&buf)
		if err := encoder.Encode(p); err != nil {
			logrus.Debugf("encoding policy %s: %s, ", name, err)
			return err
		}
		value := hex.EncodeToString(buf.Bytes())
		enforcer.policyStore.Put(name, value)
	}
	logrus.Debug("policy not existed, skip putting")
	return nil
}

func (enforcer *Enforcer) LoadPolicy(name string) error {
	enforcer.policyLock.Lock()
	defer enforcer.policyLock.Unlock()
	if _, ok := enforcer.policies[name]; !ok {
		value := enforcer.policyStore.Get(name)
		if value == "" {
			logrus.Debugf("empty policy %s", name)
			return errors.New(
				fmt.Sprintf("fail to load policy %s, no content", name))
		}
		jsonbytes, err := hex.DecodeString(value)
		if err != nil {
			logrus.Debugf("decoding hex policy %s: %s", name, err)
			return err
		}

		buf := bytes.NewBuffer(jsonbytes)
		decoder := json.NewDecoder(buf)
		var p Policy
		if err := decoder.Decode(&p); err != nil {
			logrus.Debugf("decoding policy %s: %s", name, err)
			return err
		}
		enforcer.policies[name] = &p
	}
	logrus.Debug("policy existed, skip loading")
	return nil
}
