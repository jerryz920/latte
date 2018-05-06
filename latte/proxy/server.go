package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	eurosys18 "github.com/jerryz920/conferences/latte"
	"github.com/jerryz920/conferences/latte/kvstore"
	logrus "github.com/sirupsen/logrus"
)

type MetadataProxy struct {
	client *http.Client
	cache  Cache
	addr   string
}

func (r MetadataRequest) ByteBuf() (*bytes.Buffer, error) {
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(&r); err != nil {
		return nil, err
	}
	return &buf, nil
}

var (
	ipRangeMatch *regexp.Regexp = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)-(\d+)`)
	ipPortMatch  *regexp.Regexp = regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):(\d+)`)
	pidMatch     *regexp.Regexp = regexp.MustCompile(`\['([-a-zA-Z0-9_]+)'\]`)
	pStore       map[string]*Principal
)

func ParseIP2(msg string) (string, int, int, int) {
	if matches := ipPortMatch.FindStringSubmatch(msg); len(matches) != 3 {
		logrus.Infof("not valid ip port: %s. Give up", msg)
		return "", -1, -1, http.StatusBadRequest
	} else {
		var p1 int64
		var err error
		if p1, err = strconv.ParseInt(matches[2], 10, 32); err != nil {
			logrus.Errorf("error parsing port min: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		return matches[1], int(p1), -1, http.StatusOK
	}

}

func ParseIP3(msg string) (string, int, int, int) {
	if matches := ipRangeMatch.FindStringSubmatch(msg); len(matches) != 4 {
		logrus.Infof("not valid ip:port-range: %s. Try ip-port", msg)
		return ParseIP2(msg)
	} else {
		var p1, p2 int64
		var err error
		if p1, err = strconv.ParseInt(matches[2], 10, 32); err != nil {
			logrus.Errorf("error parsing port min: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		if p2, err = strconv.ParseInt(matches[3], 10, 32); err != nil {
			logrus.Errorf("error parsing port max: %v", err)
			return "", 0, 0, http.StatusBadRequest
		}
		return matches[1], int(p1), int(p2), http.StatusOK
	}
}

func ParseIPNew(msg string) (net.IP, int, int, int) {
	ipstr, p1, p2, status := ParseIP3(msg)
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, -1, -1, http.StatusBadRequest
	}
	return ip, p1, p2, status
}

func SetCommonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func ReadRequest(r *http.Request) (*MetadataRequest, int) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("error reading the body %v\n", err)
		return nil, http.StatusBadRequest
	}
	logrus.Debug("request body = ", string(data))
	buf := bytes.NewBuffer(data)
	d := json.NewDecoder(buf)
	mr := MetadataRequest{
		targetType: NORMAL_INSTANCE_TYPE,
		targetCidr: nil,
	}
	if err := d.Decode(&mr); err != nil {
		logrus.Errorf("error decoding the body %v\n", err)
		return nil, http.StatusBadRequest
	} else {
		return &mr, http.StatusOK
	}
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
	return addr + api
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metadata Service Proxy"))
}

func main() {

	flag.Parse()
	args := flag.Args()
	addr := ""
	if len(args) < 1 {
		logrus.Info("no server address provided, debug mode")
	} else {
		addr = args[0]
	}

	logrus.SetLevel(logrus.InfoLevel)
	riakaddr := "localhost:8087"
	if len(args) >= 2 {
		riakaddr = args[1]
	}

	if len(args) >= 3 {
		//eurosys18.RestartStore(true)
		logrus.SetLevel(logrus.DebugLevel)
	}
	formatter := new(logrus.TextFormatter)
	formatter.DisableLevelTruncation = false
	formatter.FullTimestamp = false
	formatter.TimestampFormat = "1 2 3:4:5.999999"
	logrus.SetFormatter(formatter)

	riakClient := NewRiakConn()
	if err := riakClient.Connect(riakaddr); err != nil {
		logrus.Errorf("can not connect to riak address: %s, %s", riakaddr, err)
		os.Exit(1)
	}
	logrus.Info("Riak connected! Starting the API server")
	client := MetadataProxy{
		client: &http.Client{
			Transport: &http.Transport{
				DisableCompression: true,
			},
		},
		cache: NewCache(riakClient),
		addr:  addr,
	}
	server := kvstore.NewKvStore(rootHandler)

	SetupNewAPIs(&client, server)
	//// New APIs

	if err := server.ListenAndServe(eurosys18.MetadataProxyAddress); err != nil {
		logrus.Fatal("can not listen on address: ", err)
	}
}
