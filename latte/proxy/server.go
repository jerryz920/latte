package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"

	eurosys18 "github.com/jerryz920/utils/random/conferences/eurosys18"
	kvstore "github.com/jerryz920/utils/random/conferences/eurosys18/kvstore"
	log "github.com/sirupsen/logrus"
)

var (
	debugmode bool = false
)

type Principal struct {
	Name    string
	ImageID string
	IP      string
	Config  string
	PortMin int
	PortMax int
}

func (p *Principal) Serialize() []string {
	return []string{p.Name, p.ImageID, p.IP,
		fmt.Sprintf("%d", p.PortMin),
		fmt.Sprintf("%d", p.PortMax),
		p.Config}

}

func (p *Principal) ToSafeValues() []string {
	return []string{p.Name, p.ImageID, "image",
		fmt.Sprintf("%s:%d-%d", p.IP, p.PortMin, p.PortMax),
		p.Config}
}

func logHeader(resp *http.Response) {
	for k, v := range resp.Header {
		log.Debugf("header %s: %s", k, v)
	}
}

func ParsePrincipal(data []string) (*Principal, error) {
	if len(data) != 6 {
		log.Error("recovering principal from data ", data)
		return nil, errors.New("wrong number of fields")
	}
	var p Principal
	p.Name = data[0]
	p.ImageID = data[1]
	p.IP = data[2]
	if n, err := fmt.Sscan(data[3], &p.PortMin); err != nil || n != 1 {
		log.Error("recovering principal from data ", data)
		log.Error("can not parse the portMin field: ")
		return nil, errors.New("parse error")
	}
	if n, err := fmt.Sscan(data[4], &p.PortMax); err != nil || n != 1 {
		log.Error("recovering principal from data ", data)
		log.Error("can not parse the portMax field: ")
		return nil, errors.New("parse error")
	}
	p.Config = data[5]
	return &p, nil
}

type MetadataRequest struct {
	Principal   string   `json:"principal"`
	OtherValues []string `json:"otherValues"`
}

func (r MetadataRequest) ByteBuf() (*bytes.Buffer, error) {
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(&r); err != nil {
		return nil, err
	}
	return &buf, nil
}

type PrincipalResponse struct {
	Message string
}

var (
	ipRangeMatch *regexp.Regexp = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)-(\d+)`)
	ipPortMatch  *regexp.Regexp = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)`)
	pidMatch     *regexp.Regexp = regexp.MustCompile(`\['([-a-zA-Z0-9_]+)'\]`)
	pStore       map[string]*Principal
)

func ParseIP(msg string) (string, int, int, int) {
	if matches := ipRangeMatch.FindStringSubmatch(msg); len(matches) != 4 {
		log.Errorf("not valid principal ip-port range: %s", msg)
		return "", 0, 0, http.StatusBadRequest
	} else {
		var p1, p2 int64
		var err error
		if p1, err = strconv.ParseInt(matches[2], 10, 32); err != nil {
			log.Errorf("error parsing port min: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		if p2, err = strconv.ParseInt(matches[3], 10, 32); err != nil {
			log.Errorf("error parsing port max: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		return matches[1], int(p1), int(p2), http.StatusOK
	}
}

func SetCommonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func ReadRequest(r *http.Request) (*MetadataRequest, []byte, int) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("error reading the body %v\n", err)
		return nil, nil, http.StatusBadRequest
	}
	log.Debug("request body = ", string(data))
	buf := bytes.NewBuffer(data)
	d := json.NewDecoder(buf)
	mr := MetadataRequest{}
	if err := d.Decode(&mr); err != nil {
		log.Errorf("error decoding the body %v\n", err)
		return nil, nil, http.StatusBadRequest
	} else {
		return &mr, data, http.StatusOK
	}
}

func GetPrincipalID(resp *http.Response) (string, error) {
	if resp.StatusCode != http.StatusOK {
		log.Debugf("error getting response %v\n", resp.StatusCode)
		return "", errors.New("error state " + resp.Status)
	}
	decoder := json.NewDecoder(resp.Body)
	pr := PrincipalResponse{}

	if err := decoder.Decode(&pr); err != nil {
		log.Debugf("error in decoding %v\n", err)
		return "", err
	}
	/// message is in ['<ID>'] form
	var matches []string
	if matches = pidMatch.FindStringSubmatch(pr.Message); len(matches) != 2 {
		log.Debugf("error finding PID: %v", pr.Message)
		return "", errors.New("error finding pid in response")
	}
	return matches[1], nil
}

type MetadataProxy struct {
	client    *http.Client
	pmap      *Pmap
	store     eurosys18.Store
	safestore eurosys18.Store
	addr      string
}

func (c *MetadataProxy) getUrl(api string) string {
	addr := ""
	if !strings.HasSuffix(c.addr, "/") {
		addr += c.addr + "/"
	}
	if !strings.HasPrefix(c.addr, "http://") {
		addr = "http://" + addr
	}
	if strings.HasPrefix(api, "/") {
		api = api[1:]
	}
	log.Info("posting to ", addr+api)
	return addr + api
}

func (c *MetadataProxy) postInstanceSet(w http.ResponseWriter, r *http.Request) {
	m, data, status := ReadRequest(r)
	SetCommonHeader(w)
	if status != http.StatusOK {
		log.Debug("error reading request: ", status)
		return
	}

	ip, p1, p2, status := ParseIP(m.OtherValues[3])
	if status != http.StatusOK {
		log.Debug("error parsing the IP address: ", status)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error parsing the IP address"))
		return
	}

	var pid string
	if !debugmode {
		resp, err := c.client.Post(c.getUrl("/postInstanceSet"), "application/json",
			bytes.NewBuffer(data))
		if err != nil {
			log.Error("error proxying post instance set", err)
			if resp == nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
			return
		}
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			return
		}

		pid, err = GetPrincipalID(resp)
		if err != nil {
			log.Debug("error processing response: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("\"Error processing proxy response\""))
			return
		}
	} else {
		pid = m.OtherValues[0]
	}
	p := Principal{
		Name:    m.OtherValues[0],
		ImageID: m.OtherValues[1],
		IP:      ip,
		Config:  m.OtherValues[4],
		PortMin: p1,
		PortMax: p2,
	}
	//overwrite it!
	c.pmap.CreatePrincipal(ip, p1, p2, pid)
	c.store.PutValues(pid, p.Serialize())
	c.safestore.Put(m.OtherValues[0], pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"message\": \"['%s']\"}\n", pid)))
}

func (c *MetadataProxy) retractInstanceSet(w http.ResponseWriter, r *http.Request) {
	m, _, status := ReadRequest(r)
	SetCommonHeader(w)
	if status != http.StatusOK {
		log.Debug("error reading request: ", status)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	storedpid := c.safestore.Get(m.OtherValues[0])
	if storedpid == "" {
		log.Debug("principal id not found: ", m.OtherValues[0])
		w.WriteHeader(http.StatusNotFound)
		return
	}
	serialized := c.store.GetValues(storedpid)
	storedPrincipal, err := ParsePrincipal(serialized)
	if err != nil {
		log.Debug("stored principal id not found: ", storedpid)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ip, p1, p2 := storedPrincipal.IP, storedPrincipal.PortMin,
		storedPrincipal.PortMax

	newReqBody := MetadataRequest{
		Principal:   m.Principal,
		OtherValues: storedPrincipal.ToSafeValues(),
	}
	newbuf, err := newReqBody.ByteBuf()
	if err != nil {
		log.Debug("error generating the new request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pid string
	if !debugmode {
		log.Debug("outgoing delete request: ", newbuf.String())
		resp, err := c.client.Post(c.getUrl("/retractInstanceSet"), "application/json",
			newbuf)
		if err != nil {
			log.Debug("error proxying post instance set")
			if resp == nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
			return
		}
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			return
		}
		if log.GetLevel() == log.DebugLevel {
			dumpdata, _ := httputil.DumpResponse(resp, true)
			log.Debug("response data dump: ", string(dumpdata))
		}

	} else {
		pid = storedpid
	}
	c.store.Del(storedpid)
	c.safestore.Del(m.OtherValues[0])
	c.pmap.DeletePrincipal(ip, p1, p2)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"message\": \"['%s']\"}\n", pid)))
}

func (c *MetadataProxy) proxyAll(w http.ResponseWriter, r *http.Request) {
	if log.GetLevel() == log.DebugLevel {
		data, _ := httputil.DumpRequest(r, true)
		log.Debug("request data = ", string(data))
	}

	outreq, err := http.NewRequest(r.Method, c.getUrl(r.URL.RequestURI()), r.Body)

	SetCommonHeader(w)
	if err != nil {
		log.Debug("error creating new http request: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !debugmode {
		resp, err := c.client.Do(outreq)
		if err != nil {
			log.Debug("error processing proxy request: ", err)
			if resp == nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
			return
		}
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			return
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Debug("error reading the response from server: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(data)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("debugging proxy! all good\n"))
	}
}

type BearerMetadataRequest struct {
	Principal   string   `json:"principal"`
	BearerRef   string   `json:"bearerRef"`
	OtherValues []string `json:"otherValues"`
}

func (c *MetadataProxy) lookupAuthID(endorsee string) (string, string, int, error) {
	matches := ipPortMatch.FindStringSubmatch(endorsee)
	if len(matches) != 3 {
		log.Debug("error parsing the IP:port, actual", endorsee)
		return "", "", 0, errors.New("error parsing IP:Port pair")
	}
	ip := matches[1]
	port, err := strconv.ParseInt(matches[2], 10, 32)
	if err != nil {
		log.Debug("error parsing the port int")
		return "", "", 0, err
	}
	index, err := c.pmap.GetIndex(ip, int(port))
	if index != nil {
		return fmt.Sprintf("%s:%d-%d", ip, index.Pmin, index.Pmax-1),
			ip, int(port), nil
	} else {
		return "", "", 0, err
	}
}

func (c *MetadataProxy) lookupGroupPortAndBearer(endorsee string) (string, string, int, error) {
	matches := ipPortMatch.FindStringSubmatch(endorsee)
	if len(matches) != 3 {
		log.Debug("error parsing the IP:port, actual", endorsee)
		return "", "", 0, errors.New("error parsing IP:Port pair")
	}
	ip := matches[1]
	port, err := strconv.ParseInt(matches[2], 10, 32)
	if err != nil {
		log.Debug("error parsing the port int")
		return "", "", -1, err
	}
	index, err := c.pmap.GetIndex(ip, int(port))
	if index != nil {
		return index.GroupP, ip, index.GroupPort, nil
	} else {
		return "", "", 0, err
	}
}

func (c *MetadataProxy) marshalBearer(r *MetadataRequest) (*bytes.Buffer, error) {
	br := BearerMetadataRequest{
		Principal:   r.Principal,
		OtherValues: r.OtherValues,
	}

	matches := ipPortMatch.FindStringSubmatch(r.OtherValues[0])
	if len(matches) != 3 {
		log.Debug("error parsing the IP:port, actual", r.OtherValues[0])
		return nil, errors.New("error parsing IP:Port pair")
	}
	ip := matches[1]
	port, err := strconv.ParseInt(matches[2], 10, 32)
	if err != nil {
		log.Debug("error parsing the port int")
		return nil, err
	}

	pid, err := c.pmap.GetPrincipal(ip, int(port))
	if err != nil {
		log.Debugf("error searching for principal on %s:%d", ip, port)
		return nil, err
	}
	br.BearerRef = pid

	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(&br); err != nil {
		log.Debug("error encoding the principal ", err, br)
		return nil, err
	}
	return &buf, nil
}

func (c *MetadataProxy) attest(w http.ResponseWriter, r *http.Request) {
	m, _, status := ReadRequest(r)
	SetCommonHeader(w)
	if status != http.StatusOK {
		log.Debug("error reading request: ", status)
		w.WriteHeader(status)
		return
	}

	newBuf, err := c.marshalBearer(m)
	if err != nil {
		log.Debug("error finding bearer for incoming request ", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if debugmode {
		w.WriteHeader(http.StatusOK)
		content := strings.Replace(newBuf.String(), "\"", "\\\"", -1)
		content = strings.Trim(content, "\n")
		w.Write([]byte(fmt.Sprintf("{\"message\": \"%s\"}", content)))
		return
	}
	log.Debugf("bearer request body: %s\n", newBuf.String())

	outreq, err := http.NewRequest(r.Method, c.getUrl(r.URL.String()), newBuf)
	if err != nil {
		log.Debug("error creating new http request: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := c.client.Do(outreq)
	if err != nil {
		log.Debug("error processing proxy request: ", err)
		if resp != nil {
			w.WriteHeader(resp.StatusCode)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		return
	}
	logHeader(resp)

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debug("error reading the response from server: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}

func (c *MetadataProxy) postWorkerSet(w http.ResponseWriter, r *http.Request) {
	m, data, status := ReadRequest(r)
	SetCommonHeader(w)
	if status != http.StatusOK {
		log.Debug("error reading request: ", status)
		w.WriteHeader(status)
		return
	}
	/// first check if the remote principal actually exists
	authID, ip, port, err := c.lookupAuthID(m.OtherValues[1])
	if authID == "" || err != nil {
		log.Debug("can not read out endorsee ", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var pid string
	if !debugmode {
		resp, err := c.client.Post(c.getUrl("/postWorkerSet"), "application/json",
			bytes.NewBuffer(data))
		if err != nil {
			log.Error("error proxying post worker set", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			return
		}

		pid, err = GetPrincipalID(resp)
		if err != nil {
			log.Debug("error processing response: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error processing proxy response"))
			return
		}

		// update the subject token here. This is a workaround to safe's API design
		// as we don't have the exact port range for the endorsed party.
		updateReq := MetadataRequest{
			Principal:   authID,
			OtherValues: []string{pid},
		}
		buf, err := updateReq.ByteBuf()
		if err != nil {
			log.Debug("error parsing request principal: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error processing proxy response"))
			return
		}
		log.Debug("subject set req: ", buf.String())
		resp, err = c.client.Post(c.getUrl("/updateSubjectSet"), "application/json",
			buf)

		if err != nil {
			log.Error("error updating worker subject set", err)
			if resp == nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
			return
		}
		log.Info("Updating subject set, status: ", resp.StatusCode)
		pid, err = GetPrincipalID(resp)
		if err != nil {
			log.Debug("error processing response: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error processing proxy response"))
			return
		}

		if resp.StatusCode == http.StatusOK {
			c.pmap.SetPrincipalGroupPort(ip, port, pid)
		}
	} else {
		pid = m.OtherValues[0]
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"message\": \"['%s']\"}\n", pid)))
}

func (c *MetadataProxy) workerAccessesObject(w http.ResponseWriter, r *http.Request) {
	m, _, status := ReadRequest(r)
	SetCommonHeader(w)
	if status != http.StatusOK {
		log.Debug("error reading request: ", status)
		w.WriteHeader(status)
		return
	}

	bearer, ip, groupPort, err := c.lookupGroupPortAndBearer(m.OtherValues[0])
	if err != nil {
		log.Debug("error finding the group port of the principal, return reject")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	/// fix the request
	m.OtherValues[0] = fmt.Sprintf("%s:%d", ip, groupPort)
	br := BearerMetadataRequest{
		Principal:   m.Principal,
		OtherValues: m.OtherValues,
	}
	br.BearerRef = bearer

	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(&br); err != nil {
		log.Debug("error encode the bearer request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if debugmode {
		w.WriteHeader(http.StatusOK)
		buf.WriteString("{\"message\": \"debug mode\"}\n")
		w.Write(buf.Bytes())
		return
	}
	log.Debugf("converted request body: %s\n", buf.String())

	outreq, err := http.NewRequest(r.Method, c.getUrl(r.URL.String()), &buf)
	if err != nil {
		log.Debug("error creating new http request: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := c.client.Do(outreq)
	if err != nil {
		log.Debug("error sending request ", err)
		if resp != nil {
			w.WriteHeader(resp.StatusCode)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	logHeader(resp)

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debug("error reading the response from server: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(data)

}

func (c *MetadataProxy) RecoverPrincipals() {
	for _, pid := range c.store.Keys() {
		values := c.store.GetValues(pid)
		p, err := ParsePrincipal(values)
		if err != nil {
			log.Errorf("fail to recover pricipal %s: %v", pid, err)
			continue
		}
		c.pmap.CreatePrincipal(p.IP, p.PortMin, p.PortMax, pid)
	}
}

func main() {

	flag.Parse()
	args := flag.Args()
	addr := ""
	if len(args) < 1 {
		log.Info("no server address provided, debug mode")
		debugmode = true
	} else {
		addr = args[0]
	}
	if len(args) >= 2 {
		log.SetLevel(log.DebugLevel)
	}
	client := MetadataProxy{
		client: &http.Client{
			Transport: &http.Transport{
				DisableCompression: true,
			},
		},
		pmap: NewPmap(),
		addr: addr,
	}

	if len(args) >= 3 {
		eurosys18.RestartStore(true)
	}
	store, err := eurosys18.NewStore("pmap", false)
	if err != nil {
		log.Fatal("can not create pmap store ", err)
	}
	safestore, err := eurosys18.NewStore("safe", false)
	if err != nil {
		log.Fatal("can not create safe store ", err)
	}

	client.store = store
	client.safestore = safestore
	/// recover existing principals
	client.RecoverPrincipals()

	server := kvstore.NewKvStore(client.proxyAll)
	server.AddRoute("/postInstanceSet", client.postInstanceSet, "")
	server.AddRoute("/retractInstanceSet", client.retractInstanceSet, "")
	server.AddRoute("/updateSubjectSet", client.proxyAll, "")
	server.AddRoute("/postAttesterImage", client.proxyAll, "")
	server.AddRoute("/postObjectAcl", client.proxyAll, "")
	server.AddRoute("/postImageProperty", client.proxyAll, "")
	server.AddRoute("/attestInstance", client.attest, "")
	server.AddRoute("/attestAppProperty", client.attest, "")
	server.AddRoute("/appAccessesObject", client.attest, "")
	server.AddRoute("/postWorkerSet", client.postWorkerSet, "")
	server.AddRoute("/workerAccessesObject", client.workerAccessesObject, "")
	server.AddRoute("/checkImgProperty", client.proxyAll, "")

	if err := server.ListenAndServe(eurosys18.MetadataProxyAddress); err != nil {
		log.Fatal("can not listen on address: ", err)
	}
}
