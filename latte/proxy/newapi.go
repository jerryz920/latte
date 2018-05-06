package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	jhttp "github.com/jerryz920/utils/goutils/http"
	"github.com/sirupsen/logrus"
)

const (
	IaaSProvider = "152.3.145.38:444"
)

func NotImplemented(w http.ResponseWriter, r *http.Request) {
	logrus.Error("Method not implemented!")
}

func (c *MetadataProxy) authenticate(principal string) (*CachedInstance, int) {
	ip, lport, _, ok := ParseIPNew(principal)
	if ok != http.StatusOK {
		logrus.Debug("Principal looks like an UUID")
		/// Try UUID
		cachedInstance, status := c.cache.GetInstanceFromID(principal)
		if status != http.StatusOK {
			logrus.Error("fail to authenticate the UUID")
			return nil, status
		}
		return &cachedInstance, status
	} else {
		cachedInstance, status := c.cache.GetInstanceFromNetMap(ip, lport)
		if status != http.StatusOK {
			logrus.Error("fail to authenticate the network address")
			return nil, status
		}
		return &cachedInstance, status
	}
}

func (c *MetadataProxy) newAuth(r *http.Request, authPrincipal bool) (*MetadataRequest, int) {
	mr, status := ReadRequest(r)
	mr.method = r.Method
	mr.url = r.URL.RequestURI()
	if status != http.StatusOK {
		logrus.Error("error reading request in newAuth")
		return nil, status
	}
	if mr.Principal == IaaSProvider || !authPrincipal {
		mr.Principal = IaaSProvider
		return mr, status
	}

	cachedInstance, status := c.authenticate(mr.Principal)
	if status != http.StatusOK {
		logrus.Error("fail to authenticate the principal field")
		return nil, status
	}

	remoteIp, remotePort, _, status := ParseIPNew(r.RemoteAddr)
	if status != http.StatusOK {
		logrus.Error("fail to parse remote address in request: Must be a bug")
		return nil, status
	}
	if remoteIp.Equal(cachedInstance.Ip) || remotePort < cachedInstance.Lport ||
		remotePort > cachedInstance.Rport {
		////Not rejecting the request for experiments.
		logrus.Error("incoming port is not within the range.")
	}

	mr.Principal = cachedInstance.ID.Pid
	mr.ParentBear = cachedInstance.ID.PPid
	mr.ip = cachedInstance.Ip
	mr.lport = cachedInstance.Lport
	mr.rport = cachedInstance.Rport
	mr.cache = cachedInstance
	return mr, http.StatusOK
}

//// authorize if a control message will be sent by the metadata service
/// the targetIp, targetLport, targetRport is to be checked
/// the allowUUID field allows removing instances using its UUID directly
func (c *MetadataProxy) authzControl(mr *MetadataRequest, targetAddr string, allowUUID bool) int {
	var ok int
	/// Still parse the address
	mr.targetIp, mr.targetLport, mr.targetRport, ok = ParseIPNew(targetAddr)

	if ok != http.StatusOK {
		if allowUUID {
			logrus.Infof("Can not parse target field as address, trying UUID")
			cachedInstance, status := c.cache.GetInstanceFromID(targetAddr)
			if status != http.StatusOK {
				logrus.Error("fail to authenticate the UUID")
				return status
			}
			mr.targetIp, mr.targetLport, mr.targetRport =
				cachedInstance.Ip, cachedInstance.Lport, cachedInstance.Rport
		} else {
			logrus.Errorf("error parsing target IP %s", targetAddr)
			return ok
		}
	}

	if mr.Principal == IaaSProvider {
		return http.StatusOK
	}

	if mr.ip.Equal(mr.targetIp) && mr.lport <= mr.targetLport && mr.rport >= mr.targetRport {
		return http.StatusOK
	}
	// last resort: check if the ip is inside some allocation
	if alloc := mr.cache.ID.Cidr; mr.cache.ID.Type == VM_INSTANCE_TYPE && alloc != nil {
		if alloc.Contains(mr.targetIp) {
			return http.StatusOK
		}
	}
	return http.StatusUnauthorized
}

func (c *MetadataProxy) preInstanceCallHandler(mr *MetadataRequest) (string, int) {
	if status := c.authzControl(mr, mr.OtherValues[2], false); status != http.StatusOK {
		return fmt.Sprintf("can not authorize request: %s, %s:%d-%d to %s:%d-%d\n",
				mr.Principal, mr.ip, mr.lport, mr.rport,
				mr.targetIp, mr.targetLport, mr.targetRport),
			status
	}

	/// AuthID is not used any more, remove it
	newRequest := append(mr.OtherValues[:2], mr.OtherValues[3:]...)
	mr.OtherValues = newRequest

	return "", http.StatusOK
}

func (c *MetadataProxy) postInstanceCreationHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	if status != http.StatusOK {
		return string(data), status
	}

	instance := &CachedInstance{
		Ip:    mr.targetIp,
		Lport: mr.targetLport,
		Rport: mr.targetRport,
		ID: &InstanceCred{
			Pid:  mr.OtherValues[0],
			PPid: mr.Principal,
			Type: mr.targetType,
			Cidr: mr.targetCidr,
		},
	}
	c.cache.PutInstance(instance)
	return fmt.Sprintf("{\"message\": \"['%s']\"}\n", mr.OtherValues[0]),
		http.StatusOK
}

func (c *MetadataProxy) postInstanceDeletionHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	if status != http.StatusOK {
		return string(data), status
	}

	c.cache.DelInstance(mr.targetIp, mr.targetLport, mr.targetRport, mr.OtherValues[0])
	return fmt.Sprintf("{\"message\": \"['%s']\"}\n", mr.OtherValues[0]),
		http.StatusOK
}

func (c *MetadataProxy) preVMInstanceCallHandler(mr *MetadataRequest) (string, int) {
	if mr.Principal != IaaSProvider {
		return "only IaaS provider can create VM", http.StatusUnauthorized
	}
	_, cidr, err := net.ParseCIDR(mr.OtherValues[3])
	if err != nil {
		msg := fmt.Sprintf("can not allocate CIDR %s\n", err)
		return msg, http.StatusBadRequest
	}
	mr.targetCidr = cidr
	mr.targetType = VM_INSTANCE_TYPE
	/// Cidr is no longer used
	newRequest := append(mr.OtherValues[:3], mr.OtherValues[4:]...)
	mr.OtherValues = newRequest

	return c.preInstanceCallHandler(mr)
}

func (c *MetadataProxy) preLegacyInstanceCallHandler(mr *MetadataRequest) (string, int) {

	if mr.Principal == IaaSProvider {
		// Creating VM instance
		msg := fmt.Sprintf("Should not use legacy API to create VMs!")
		return msg, http.StatusBadRequest
	}

	newRequest := make([]string, 2)
	newRequest[0] = mr.OtherValues[0]
	newRequest[1] = mr.OtherValues[1]
	mr.OtherValues = newRequest
	return c.preInstanceCallHandler(mr)
}

func (c *MetadataProxy) createInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createInstanceLegacy(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preLegacyInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createVMInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preVMInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) deleteInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preInstanceCallHandler,
		c.postInstanceDeletionHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) deleteVMInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preVMInstanceCallHandler,
		c.postInstanceDeletionHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createImageLink(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		creatorInstance, status := c.authenticate(mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("cannot authenticate Image creator\n"), status
		}
		mr.OtherValues[0] = creatorInstance.ID.Pid
		return "", http.StatusOK
	}, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) lazyDeleteInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r,
		func(mr *MetadataRequest) (string, int) {
			if status := c.authzControl(mr, mr.OtherValues[0], true); status != http.StatusOK {
				return fmt.Sprintf("can not authorize request: %s, %s:%d-%d to %s:%d-%d\n",
						mr.Principal, mr.ip, mr.lport, mr.rport,
						mr.targetIp, mr.targetLport, mr.targetRport),
					status
			}
			mr.targetType = NORMAL_INSTANCE_TYPE
			mr.targetCidr = nil
			return "", http.StatusOK
		},
		c.postInstanceDeletionHandler,
		true,
	)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createMembership(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r,
		func(mr *MetadataRequest) (string, int) {
			memberInstance, status := c.authenticate(mr.OtherValues[1])
			if status != http.StatusOK {
				return fmt.Sprintf("cannot authenticate Image creator\n"), status
			}
			mr.OtherValues[1] = memberInstance.ID.Pid
			return "", http.StatusOK
		},
		nil, true,
	)

	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) createInstanceConfig(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {

		cachedInstance, status := c.authenticate(mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("Fail to authenticate the instance %s",
				mr.OtherValues[0]), status
		}
		if mr.Principal != cachedInstance.ID.PPid {
			return fmt.Sprintf("Config can only be published by host instance:"+
					"Speaker: %s, Pid: %s, Saved-PPid: %s", mr.Principal,
					cachedInstance.ID.Pid, cachedInstance.ID.PPid),
				http.StatusUnauthorized
		}
		mr.OtherValues[0] = cachedInstance.ID.Pid

		remain := len(mr.OtherValues) - 1
		current := 1 /// skip the instance ID.
		for remain > 10 {
			args := make([]string, 0)
			args = append(args, mr.OtherValues[0])
			args = append(args, mr.OtherValues[current:current+10]...)

			copyr := &MetadataRequest{
				Principal:   mr.Principal,
				OtherValues: args,
				method:      "POST",
				url:         "/postInstanceConfig5",
			}
			msg, status := c.newHandler(copyr, nil, nil)

			if status != http.StatusOK {
				logrus.Error("error posting configurations, original: %v, left: %v",
					mr.OtherValues, mr.OtherValues[current:])
				return msg, status
			}
			current += 10
			remain -= 10
		}
		mr.method = "POST"
		mr.url = fmt.Sprintf("%s%d", mr.url, remain/2)
		/// The handler will continue to handle the remaining configs
		return "", http.StatusOK
	}, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) handleCheckInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		/// convert otherValues[0] to uuid
		cache, status := c.authenticate(mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("target not found %s", mr.OtherValues[0]), status
		}
		mr.OtherValues[0] = cache.ID.Pid
		mr.ParentBear = cache.ID.PPid
		return "", http.StatusOK
	}, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) handleOther(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, nil, nil, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) newHandlerUnwrapped(r *http.Request, preHook func(*MetadataRequest) (string, int),
	postHook func(*MetadataRequest, []byte, int) (string, int), authPrincipal bool) (string, int) {
	/// FIXME: authparent is no longer used
	metareq, status := c.newAuth(r, authPrincipal)
	if status != http.StatusOK {
		return "can not authenticate request\n", status
	}
	return c.newHandler(metareq, preHook, postHook)
}

/// A pre hook will be run before sending out request, post hook will run after
// correctly fetch the data
func (c *MetadataProxy) newHandler(mr *MetadataRequest, preHook func(*MetadataRequest) (string, int),
	postHook func(*MetadataRequest, []byte, int) (string, int)) (string, int) {

	t1 := time.Now()
	if preHook != nil {
		msg, status := preHook(mr)
		if status != http.StatusOK {
			return msg, status
		}
	}
	t2 := time.Now()

	buf, err := EncodingMetadataRequest(mr)
	if err != nil {
		msg := fmt.Sprintf("error encoding mr %v\n", err)
		return msg, http.StatusInternalServerError
	}
	logrus.Debug("MR =", buf.String())

	outreq, err := http.NewRequest(mr.method, c.getUrl(mr.url), buf)
	if err != nil {
		msg := fmt.Sprintf("fail to generate new request %v\n", err)
		return msg, http.StatusInternalServerError
	}

	resp, err := c.client.Do(outreq)
	if err != nil {
		msg := fmt.Sprintf("error proxying post instance set % v\n", err)
		if resp == nil {
			return msg, http.StatusInternalServerError
		} else {
			return msg, resp.StatusCode
		}
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("error reading the response from server: %v\n", err)
		return msg, http.StatusInternalServerError
	}
	logrus.Debugf("Real response of %s: %s", mr.url, string(data))
	t3 := time.Now()

	if postHook != nil {
		msg, status := postHook(mr, data, resp.StatusCode)
		t4 := time.Now()
		logrus.Infof("PERF %s %f %f %f", mr.url, t2.Sub(t1).Seconds(), t3.Sub(t2).Seconds(), t4.Sub(t3).Seconds())
		return msg, status
	}
	logrus.Infof("PERF %s %f %f", mr.url, t2.Sub(t1).Seconds(), t3.Sub(t2).Seconds())
	// We allow call another http method in posthook, however, its result
	// is consumed internally only
	return string(data), resp.StatusCode
}

func SetupNewAPIs(c *MetadataProxy, server *jhttp.APIServer) {

	server.AddRoute("/postInstance", c.createInstance, "")
	server.AddRoute("/postInstanceSet", c.createInstanceLegacy, "")
	server.AddRoute("/retractInstanceSet", c.deleteInstance, "")
	server.AddRoute("/postVMInstance", c.createVMInstance, "")
	server.AddRoute("/postLinkImageOwner", c.createImageLink, "")
	server.AddRoute("/delInstance", c.deleteInstance, "")
	server.AddRoute("/delVMInstance", c.deleteVMInstance, "")
	server.AddRoute("/lazyDeleteInstance", c.lazyDeleteInstance, "")
	/// Handling any num of configs
	server.AddRoute("/postInstanceConfig", c.createInstanceConfig, "")

	/// The proxy to do here is similar, just authenticate the IDs
	server.AddRoute("/postAckMembership", c.createMembership, "")
	server.AddRoute("/postMembership", c.createMembership, "")

	otherMethods := []string{
		"/postCluster",
		"/delAckMembership",
		"/delCluster",
		"/delConditionalEndorsement",
		"/delEndorsement",
		"/delInstanceAuthID",
		"/delInstanceAuthKey",
		"/delInstanceCidrConfig",
		"/delInstanceConfig1",
		"/delInstanceConfig2",
		"/delInstanceConfig3",
		"/delInstanceConfig4",
		"/delInstanceConfig5",
		"/delInstanceControl",
		"/delLinkImageOwner",
		"/delMembership",
		"/delParameterizedConnection",
		"/delParameterizedEndorsement",
		"/delVpcConfig1",
		"/delVpcConfig2",
		"/delVpcConfig3",
		"/delVpcConfig4",
		"/delVpcConfig5",
		"/postConditionalEndorsement",
		"/postEndorsement",
		//"/postInstanceAuthID",
		//"/postInstanceAuthKey",
		//"/postInstanceCidrConfig",
		"/postInstanceConfig1",
		"/postInstanceConfig2",
		"/postInstanceConfig3",
		"/postInstanceConfig4",
		"/postInstanceConfig5",
		//"/postInstanceControl",
		"/postParameterizedConnection",
		"/postParameterizedEndorsement",
		"/postVpcConfig1",
		"/postVpcConfig2",
		"/postVpcConfig3",
		"/postVpcConfig4",
		"/postVpcConfig5",
	}

	checkMethods := []string{
		"/checkClusterGuard",
		"/checkCodeQuality",
		"/checkContainerIsolation",
		"/checkFetch",
		//	"/checkTrustedCode", Should use some other check
		"/checkTrustedConnections",
	}

	for _, method := range checkMethods {
		server.AddRoute(method, c.handleCheckInstance, "")
	}
	for _, method := range otherMethods {
		server.AddRoute(method, c.handleOther, "")
	}

	return

}
