package main

import (
	"bytes"
	"encoding/json"
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

func EncodingMetadataRequest(mr *MetadataRequest) (*bytes.Buffer, error) {
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(mr); err != nil {
		logrus.Debug("error encoding the principal ", err)
		return nil, err
	}
	return &buf, nil
}

func (c *MetadataProxy) reloadCache(ip net.IP) int {
	if ip == nil {
		logrus.Error("can not parse given IP address")
		return http.StatusBadRequest
	}
	pmaps, err := c.newstore.GetAllNetID(ip)
	if err != nil {
		logrus.Error("can not authenticate the IP address, ", err)
		return http.StatusUnauthorized
	}
	for _, m := range pmaps {
		/// FIXME: this type of operations should be abstracted uniformly
		c.pmap.CreatePrincipalPP(ip.String(), m.Lport, m.Rport, m.ID.Id, m.ID.Pid)
	}
	/// if it has ip -> cidr allocation, load it as well

	alloc, err := c.newstore.GetNetAllocation(ip)
	if err != nil {
		logrus.Error("error in get net alloc for ", ip)
		return http.StatusInternalServerError
	}
	if alloc != nil {
		c.AddAllocCache(ip, alloc)
	}
	return http.StatusOK
}

func (c *MetadataProxy) strNetToID(id string) (string, int) {
	p, _, status := c.strNetToCred(id)
	return p, status
}

func (c *MetadataProxy) strNetToCred(id string) (string, string, int) {
	if id == IaaSProvider {
		return id, id, http.StatusOK
	}
	ip, port, _, status := ParseIPNew(id)
	if status != http.StatusOK {
		logrus.Infof("can not parse AuthID from string %s, try use as UUID", id)
		return id, "", http.StatusOK
	}

	ipstr := ip.String()
	if !c.pmap.Loaded(ipstr) {
		logrus.Infof("Reloading cache for %s", ip.String())
		status := c.reloadCache(ip)
		if status != http.StatusOK {
			return "", "", status
		}
	}
	instIndex, err := c.pmap.GetIndex(ipstr, port)
	if instIndex == nil {
		logrus.Debug("error in get index for IP and port ", ip, port, err)
		return "", "", http.StatusInternalServerError
	}
	return instIndex.P, instIndex.PP, http.StatusOK
}

func (c *MetadataProxy) netToCred(ip net.IP, lport, rport int) (string, string, int) {
	//	resp, err := c.client.Post(c.getUrl("/postInstance"), "application/json", buf)
	ipstr := ip.String()
	if !c.pmap.Loaded(ipstr) {
		status := c.reloadCache(ip)
		if status != http.StatusOK {
			return "", "", status
		}
	}
	instIndex, err := c.pmap.GetIndex(ipstr, lport)
	if instIndex == nil {
		logrus.Error("error in get index for IP and port ", ip, lport, rport, err)
		return "", "", http.StatusInternalServerError
	}

	logrus.Debug("port ", instIndex.Pmin, instIndex.Pmax, lport, rport)
	if instIndex.Pmin == lport && instIndex.Pmax-1 == rport {
		return instIndex.P, instIndex.PP, http.StatusOK
	} else {
		return "", "", http.StatusUnauthorized
	}
}

func (c *MetadataProxy) createNetToID(ip net.IP, lport, rport int, uuid, puuid string) error {
	c.pmap.CreatePrincipalPP(ip.String(), lport, rport, uuid, puuid)
	return c.newstore.PutNetIDMap(ip, lport, rport, &InstanceCred{Id: uuid, Pid: puuid})
}

func (c *MetadataProxy) deleteNetToID(ip net.IP, lport, rport int) error {
	c.pmap.DeletePrincipal(ip.String(), lport, rport)
	return c.newstore.DelNetIDMap(ip, lport, rport)
}

func (c *MetadataProxy) AddAlloc(ip net.IP, cidr *net.IPNet) {
	c.pmap.CidrAlloc[ip.String()] = cidr
	c.newstore.PutNetAllocation(ip, cidr)
}

func (c *MetadataProxy) AddAllocCache(ip net.IP, cidr *net.IPNet) {
	c.pmap.CidrAlloc[ip.String()] = cidr
}

func (c *MetadataProxy) DelAlloc(ip net.IP) {
	delete(c.pmap.CidrAlloc, ip.String())
	c.newstore.DelNetAllocation(ip)
}

func (c *MetadataProxy) newAuth(r *http.Request, authParent bool) (*MetadataRequest, int) {
	mr, _, status := ReadRequest(r)
	mr.method = r.Method
	mr.url = r.URL.RequestURI()
	if status != http.StatusOK {
		logrus.Error("error reading request in newAuth")
		return nil, status
	}
	if mr.Principal == IaaSProvider {
		mr.Principal = IaaSProvider
		return mr, status
	}

	ip, lport, rport, ok := ParseIPNew(mr.Principal)
	if ok != http.StatusOK {
		logrus.Info("error parsing principal as network in newAuth. We may support direct uuid in principal field in future as authid, but not now")
		return nil, ok
	}

	uuid, puuid, status := c.netToCred(ip, lport, rport)
	if status != http.StatusOK {
		logrus.Error("converting principal to ID in newAuth")
		return nil, status
	}
	if uuid == "" {
		logrus.Error("can not find instance uuid")
		return nil, http.StatusUnauthorized
	}
	if authParent && puuid == "" {
		logrus.Error("can not find instance uuid")
		return nil, http.StatusUnauthorized
	}
	mr.Principal = uuid
	mr.ParentBear = puuid
	mr.ip = ip
	mr.lport = lport
	mr.rport = rport
	return mr, http.StatusOK
}

//// authorize if a control message will be sent by the metadata service
/// the targetIp, targetLport, targetRport is to be checked
func (c *MetadataProxy) authzControl(mr *MetadataRequest) int {
	if mr.Principal == IaaSProvider {
		return http.StatusOK
	}
	ip := mr.ip
	ipstr := ip.String()
	if !c.pmap.Loaded(ipstr) {
		status := c.reloadCache(ip)
		if status != http.StatusOK {
			logrus.Error("fail to reload netmap cache")
			return http.StatusInternalServerError
		}
	}

	if ip.Equal(mr.targetIp) && mr.lport <= mr.targetLport && mr.rport >= mr.targetRport {
		return http.StatusOK
	}

	// last resort: check if the ip is inside some allocation
	if alloc, ok := c.pmap.CidrAlloc[ipstr]; ok {
		if alloc.Contains(mr.targetIp) {
			return http.StatusOK
		}
	}
	return http.StatusUnauthorized
}

func (c *MetadataProxy) preInstanceCallHandler(mr *MetadataRequest) (string, int) {
	var ok int
	mr.targetIp, mr.targetLport, mr.targetRport, ok = ParseIPNew(mr.OtherValues[2])
	if ok != http.StatusOK {
		return fmt.Sprintf("error parsing target IP %s", mr.OtherValues[2]), ok
	}
	/// authz
	if status := c.authzControl(mr); status != http.StatusOK {
		return fmt.Sprintf("can not authorize request: %s, %s:%d-%d to %s:%d-%d\n",
				mr.Principal, mr.ip, mr.lport, mr.rport,
				mr.targetIp, mr.targetLport, mr.targetRport),
			status
	}

	/// workaround: vm creation has 6 parameter, it's on 5th location, other instance on the 4th
	var imageStoreIdx int
	if len(mr.OtherValues) == 6 {
		imageStoreIdx = 4
	} else {
		imageStoreIdx = 3
	}
	logrus.Debug("converting IP-UUID %s", mr.OtherValues[imageStoreIdx])
	imageOwnerUUID, status := c.strNetToID(mr.OtherValues[imageStoreIdx])
	if status != http.StatusOK {
		return fmt.Sprintf("can not authenticate the image store service\n"), status
	}
	mr.OtherValues[imageStoreIdx] = imageOwnerUUID
	return "", http.StatusOK
}

func (c *MetadataProxy) postInstanceCreationHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	if status != http.StatusOK {
		return string(data), status
	}

	controlMr := MetadataRequest{
		Principal:   IaaSProvider,
		OtherValues: []string{mr.Principal, mr.OtherValues[0]},
		method:      "POST",
		url:         "/postInstanceControl",
	}
	msg, controlStatus := c.newHandler(&controlMr, nil, nil)
	if controlStatus != http.StatusOK {
		return msg, controlStatus
	}
	c.createNetToID(mr.targetIp, mr.targetLport, mr.targetRport, mr.OtherValues[0], mr.Principal)
	return fmt.Sprintf("{\"message\": \"['%s']\"}\n", mr.OtherValues[0]),
		http.StatusOK
}

func (c *MetadataProxy) postInstanceDeletionHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	if status != http.StatusOK {
		return string(data), status
	}

	controlMr := MetadataRequest{
		Principal:   IaaSProvider,
		OtherValues: []string{mr.Principal, mr.OtherValues[0]},
		method:      "POST",
		url:         "/delInstanceControl",
	}
	msg, controlStatus := c.newHandler(&controlMr, nil, nil)
	if controlStatus != http.StatusOK {
		return msg, controlStatus
	}
	c.deleteNetToID(mr.targetIp, mr.targetLport, mr.targetRport)
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

	return c.preInstanceCallHandler(mr)
}

func (c *MetadataProxy) preLegacyInstanceCallHandler(mr *MetadataRequest) (string, int) {

	if mr.Principal == IaaSProvider {
		// Creating VM instance
		msg := fmt.Sprintf("Should not use legacy API to create VMs!")
		return msg, http.StatusBadRequest
	}

	/// FIXME: here is a hack to make experiments working
	/// We do not have information about who creates the image in legacy postInstanceSet
	/// call. So we use a predefined "ImageStore"
	newRequest := make([]string, 4)
	newRequest[0] = mr.OtherValues[0]
	newRequest[1] = mr.OtherValues[1]
	newRequest[2] = mr.OtherValues[3]
	/// Maybe we could reuse the 2nd field in old request
	newRequest[3] = IMAGE_STORAGE_SERVICE
	mr.OtherValues = newRequest
	return c.preInstanceCallHandler(mr)
}

func (c *MetadataProxy) postVMInstanceCreationHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	msg, newStatus := c.postInstanceCreationHandler(mr, data, status)
	/// store cidr alloc
	if newStatus == http.StatusOK {
		c.AddAlloc(mr.targetIp, mr.targetCidr)
	}
	return msg, newStatus
}

func (c *MetadataProxy) postVMInstanceDeletionHandler(mr *MetadataRequest, data []byte,
	status int) (string, int) {
	msg, newStatus := c.postInstanceDeletionHandler(mr, data, status)
	/// store cidr alloc
	if newStatus == http.StatusOK {
		c.DelAlloc(mr.targetIp)
	}
	return msg, newStatus
}

func (c *MetadataProxy) postInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) postInstanceLegacy(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preLegacyInstanceCallHandler,
		c.postInstanceCreationHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) postVMInstance(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, c.preVMInstanceCallHandler,
		c.postVMInstanceCreationHandler, false)
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
		c.postVMInstanceDeletionHandler, true)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) postLinkImageOwner(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		creator, status := c.strNetToID(mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("cannot authenticate Image creator\n"), status
		}
		mr.OtherValues[0] = creator
		return "", http.StatusOK
	}, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) handleCheck(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, func(mr *MetadataRequest) (string, int) {
		/// convert otherValues[0] to uuid
		uuid, status := c.strNetToID(mr.OtherValues[0])
		if status != http.StatusOK {
			return fmt.Sprintf("target not found %s", mr.OtherValues[0]), status
		}
		mr.OtherValues[0] = uuid
		return "", http.StatusOK
	}, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) handleOther(w http.ResponseWriter, r *http.Request) {
	SetCommonHeader(w)
	msg, status := c.newHandlerUnwrapped(r, nil, nil, false)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (c *MetadataProxy) newHandlerUnwrapped(r *http.Request, preHook func(*MetadataRequest) (string, int),
	postHook func(*MetadataRequest, []byte, int) (string, int), authParent bool) (string, int) {
	metareq, status := c.newAuth(r, authParent)
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

	server.AddRoute("/postInstance", c.postInstance, "")
	server.AddRoute("/postInstanceSet", c.postInstanceLegacy, "")
	server.AddRoute("/retractInstanceSet", c.deleteInstance, "")
	server.AddRoute("/postVMInstance", c.postVMInstance, "")
	server.AddRoute("/postLinkImageOwner", c.postLinkImageOwner, "")
	server.AddRoute("/delInstance", c.deleteInstance, "")
	server.AddRoute("/delVMInstance", c.deleteVMInstance, "")
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
		"/lazyDeleteInstance",
		"/postAckMembership",
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
		"/postInstanceControl",
		"/postMembership",
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
		"/checkTrustedCode",
		"/checkTrustedConnections",
	}

	for _, method := range checkMethods {
		server.AddRoute(method, c.handleCheck, "")
	}
	for _, method := range otherMethods {
		server.AddRoute(method, c.handleOther, "")
	}

	return

}
