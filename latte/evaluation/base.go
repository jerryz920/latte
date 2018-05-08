package latte

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
)

type MetadataRequest struct {
	Principal   string   `json:"principal"`
	OtherValues []string `json:"otherValues"`
	Auth        string   `json:"auth,omitempty"`
}

type MetadataClient struct {
	Host   string
	Client http.Client
	Index  int
}

var (
	clientIndex = 0
)

func (m *MetadataClient) Request(cmd string, principal string, otherValues ...string) error {
	tx := time.Now()
	buf := bytes.NewBuffer(nil)
	data := MetadataRequest{
		Principal:   principal,
		OtherValues: otherValues,
	}
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(&data); err != nil {
		logrus.Error("error in json encoding: ", err)
		return err
	}
	url := fmt.Sprintf("%s%s", m.Host, cmd)
	resp, err := m.Client.Post(url, "application/json", buf)
	if err != nil {
		logrus.Error("error in requesting:", err)
		return err
	}

	msg, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logrus.Error("error in readall")
		return err
	}
	ts := time.Now().Sub(tx).Seconds()
	logrus.WithField("name", cmd).WithField("speaker", principal).
		WithField("time", ts).WithField("detail", otherValues).Info(string(msg))
	return nil
}

func NewClient(addr string) *MetadataClient {
	clientIndex++
	return &MetadataClient{
		Host: addr,
		Client: http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 128,
			},
			Timeout: time.Second * 10,
		},
		Index: clientIndex,
	}
}
