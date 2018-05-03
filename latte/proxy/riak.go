package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	riak "github.com/basho/riak-go-client"
	"github.com/sirupsen/logrus"
)

type InstanceCred struct {
	Id  string `json:"Id_s"`
	Pid string `json:"Pid_s"`
}

const (
	NETMAP_PID = "Pid_s"
	NETMAP_ID  = "Id_s"
)

func InstanceCredFromBytes(data []byte) *InstanceCred {

	buf := bytes.NewBuffer(data)
	decoder := json.NewDecoder(buf)
	var inst InstanceCred
	if err := decoder.Decode(&inst); err != nil {
		logrus.Error("decoding Json")
		return nil
	}
	return &inst
}

func (i *InstanceCred) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	decoder := json.NewEncoder(buf)
	if err := decoder.Encode(i); err != nil {
		logrus.Error("encoding Json")
		return nil
	}
	return buf.Bytes()
}

type PortRangeMap struct {
	Lport int
	Rport int
	ID    *InstanceCred
}

type RiakConn interface {
	Connect(addr string) error
	PutNetAllocation(ip net.IP, cidr *net.IPNet) error
	GetNetAllocation(ip net.IP) (*net.IPNet, error)
	DelNetAllocation(ip net.IP) error
	PutNetIDMap(ip net.IP, lport int, rport int, uuid *InstanceCred) error
	GetNetIDMap(ip net.IP, lport int, rport int) (*InstanceCred, error)
	DelNetIDMap(ip net.IP, lport int, rport int) error
	GetAllNetID(ip net.IP) ([]PortRangeMap, error)
	Shutdown() error
}

type riakConn struct {
	Addr   string
	Client *riak.Client
	/// TODO: adding settings for replications
}

func NewRiakConn() RiakConn {
	//riak.EnableDebugLogging = true
	return &riakConn{Addr: "", Client: nil}
}

func NetMapKey(lport, rport int) string {
	return fmt.Sprintf("%d:%d", lport, rport)
}

func (c *riakConn) checkIndex(indexName string) error {
	cmd, err := riak.NewFetchIndexCommandBuilder().
		WithIndexName(indexName).Build()
	if err != nil {
		logrus.Debug("building the fetch index cmd ", err)
		return err
	}

	//// If no error, check if the index really exists
	if err = c.Client.Execute(cmd); err == nil {
		logrus.Debug("valiadting existing index")
		result := cmd.(*riak.FetchIndexCommand).Response
		if result != nil && len(result) > 0 && result[0].Name == indexName {
			logrus.Debugf("Index %s exists", indexName)
			return nil
		}
	}

	cmd, err = riak.NewStoreIndexCommandBuilder().
		WithIndexName(indexName).
		WithTimeout(time.Second * 10).
		Build()
	if err != nil {
		logrus.Debug("building the create index cmd ", err)
		return err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("creating index: ", err)
		return err
	}
	logrus.Debugf("Index %s created", indexName)

	cmd, err = riak.NewStoreBucketTypePropsCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithSearchIndex(RIAK_INDEX_NAME).
		WithSearch(true).Build()

	if err != nil {
		logrus.Debug("building the store bucket type property cmd ", err)
		return err
	}
	logrus.Debug("associating bucket type index")
	return c.Client.Execute(cmd)
}

func (c *riakConn) Connect(addr string) error {
	if c.Client != nil {
		if err := c.Client.Stop(); err != nil {
			logrus.Errorf("can not stop previous riak conn to ", c.Addr)
			return err
		}
	}
	options := riak.NewClientOptions{
		RemoteAddresses: []string{addr},
	}
	client, err := riak.NewClient(&options)
	if err != nil {
		logrus.Errorf("can not connect to given address %s", addr)
		return err
	}
	c.Addr = addr
	c.Client = client
	logrus.Info("Checking index on the bucket type")

	/// check the indexes on the bucket type

	return c.checkIndex(RIAK_INDEX_NAME)
}

/// This can only be called for IAAS so no need to remember speaker
func (c *riakConn) PutNetAllocation(ip net.IP, cidr *net.IPNet) error {

	t1 := time.Now()
	obj := &riak.Object{
		ContentType:     "application/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		BucketType:      RIAK_BUCKET_TYPE,
		Bucket:          CIDR_BUCKET,
		Key:             ip.String(),
		Value:           []byte(cidr.String()),
	}

	cmd, err := riak.NewStoreValueCommandBuilder().WithContent(obj).Build()
	if err != nil {
		logrus.Debug("error in building PutNetAllocation cmd")
		return err
	}

	err = c.Client.Execute(cmd)

	logrus.Info("PERFRIAK PutAlloc ", time.Now().Sub(t1).Seconds())

	return err
}

func (c *riakConn) GetNetAllocation(ip net.IP) (*net.IPNet, error) {

	t1 := time.Now()
	cmd, err := riak.NewFetchValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(CIDR_BUCKET).
		WithKey(ip.String()).
		Build()
	if err != nil {
		logrus.Debug("error in building fetch net allocation command")
		return nil, err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error executing fetch net allocation command")
		return nil, err
	}

	if actual, ok := cmd.(*riak.FetchValueCommand); ok {

		if len(actual.Response.Values) == 0 {
			logrus.Info("there is no allocation for this IP")
			return &net.IPNet{ip, net.CIDRMask(32, 32)}, nil
		} else if len(actual.Response.Values) > 1 {
			logrus.Warning("there is more than one allocation: ")
			for _, o := range actual.Response.Values {
				logrus.Info("%v", string(o.Value))
			}
		}
		netString := string(actual.Response.Values[0].Value)
		if _, network, err := net.ParseCIDR(netString); err != nil {
			logrus.Debug("error parsing CIDR ", netString)
			return nil, err
		} else {
			logrus.Info("PERFRIAK GetAlloc ", time.Now().Sub(t1).Seconds())
			return network, nil
		}

	}
	logrus.Debug("error in reading response of net allocation command")
	return nil, errors.New("Unknown command")
}

func (c *riakConn) DelNetAllocation(ip net.IP) error {
	t1 := time.Now()
	cmd, err := riak.NewDeleteValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(CIDR_BUCKET).
		WithKey(ip.String()).
		Build()
	if err != nil {
		return err
	}
	err = c.Client.Execute(cmd)
	logrus.Info("PERFRIAK DelAlloc ", time.Now().Sub(t1).Seconds())
	return err
}

func (c *riakConn) PutNetIDMap(ip net.IP, lport int, rport int, uuid *InstanceCred) error {
	t1 := time.Now()
	obj := &riak.Object{
		ContentType:     "application/json",
		Charset:         "utf-8",
		ContentEncoding: "utf-8",
		BucketType:      RIAK_BUCKET_TYPE,
		Bucket:          ip.String(),
		Key:             NetMapKey(lport, rport),
		Value:           uuid.Bytes(),
	}
	cmd, err := riak.NewStoreValueCommandBuilder().
		WithContent(obj).Build()
	if err != nil {
		logrus.Debug("error in building PutNetIDMap cmd")
		return err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error in executing PutNetIDMap")
		return err
	}
	logrus.Info("PERFRIAK PutId ", time.Now().Sub(t1).Seconds())
	return nil
}

func (c *riakConn) GetNetIDMap(ip net.IP, lport int, rport int) (*InstanceCred, error) {

	t1 := time.Now()
	cmd, err := riak.NewFetchValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(ip.String()).
		WithKey(NetMapKey(lport, rport)).
		Build()
	if err != nil {
		logrus.Debug("error in building fetch net id map command")
		return nil, err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error executing fetch net id map command")
		return nil, err
	}

	if actual, ok := cmd.(*riak.FetchValueCommand); ok {
		if len(actual.Response.Values) == 0 {
			return nil, nil
		} else if len(actual.Response.Values) > 1 {
			logrus.Error("there is more than one UUID: BUG")
			for _, o := range actual.Response.Values {
				logrus.Info("%v", string(o.Value))
			}
		}
		return InstanceCredFromBytes(actual.Response.Values[0].Value), nil
	}
	logrus.Debug("error in reading response of get net id map")
	logrus.Info("PERFRIAK GetID ", time.Now().Sub(t1).Seconds())
	return nil, errors.New("Unknown command")
}

func (c *riakConn) DelNetIDMap(ip net.IP, lport, rport int) error {
	t1 := time.Now()
	cmd, err := riak.NewDeleteValueCommandBuilder().
		WithBucketType(RIAK_BUCKET_TYPE).
		WithBucket(ip.String()).
		WithKey(NetMapKey(lport, rport)).
		Build()
	if err != nil {
		return err
	}
	err = c.Client.Execute(cmd)
	logrus.Info("PERFRIAK DelID ", time.Now().Sub(t1).Seconds())
	return err
}

func (c *riakConn) GetAllNetID(ip net.IP) ([]PortRangeMap, error) {
	t1 := time.Now()
	query := fmt.Sprintf("%s:%s AND %s:%s", QUERY_BUCKET, ip.String(),
		QUERY_BUCKET_TYPE, RIAK_BUCKET_TYPE)
	cmd, err := riak.NewSearchCommandBuilder().
		WithReturnFields(QUERY_KEY, NETMAP_PID, NETMAP_ID).
		WithIndexName(RIAK_INDEX_NAME).
		WithQuery(query).Build()

	result := make([]PortRangeMap, 0)
	if err != nil {
		logrus.Debug("error in building fetch all net id map command")
		return result, err
	}

	if err = c.Client.Execute(cmd); err != nil {
		logrus.Debug("error executing fetch all net id map command")
		return result, err
	}

	if actual, ok := cmd.(*riak.SearchCommand); ok {
		if len(actual.Response.Docs) == 0 {
			return result, nil
		}
		for _, d := range actual.Response.Docs {
			pids, ok := d.Fields[NETMAP_PID]
			if !ok || pids == nil || len(pids) == 0 {
				logrus.Error("missing ParentID in index: %s,%s, [%v]",
					d.Bucket, d.Key, d.Fields)
				continue
			}
			pid := pids[0]

			ids, ok := d.Fields[NETMAP_ID]
			if !ok || ids == nil || len(ids) == 0 {
				logrus.Error("missing ID in index: %s,%s, [%v]",
					d.Bucket, d.Key, d.Fields)
				continue
			}
			id := ids[0]
			var lport, rport int
			if n, err := fmt.Sscanf(d.Key, "%d:%d", &lport, &rport); err != nil || n != 2 {
				logrus.Error("wrong key format of index: %s,%s", d.Bucket, d.Key)
				if err != nil {
					logrus.Error("error: ", err)
				}
				continue
			}
			result = append(result, PortRangeMap{
				Lport: lport,
				Rport: rport,
				ID: &InstanceCred{
					Pid: pid,
					Id:  id,
				},
			})
		}
		logrus.Info("PERFRIAK GetAllID ", time.Now().Sub(t1).Seconds())
		return result, nil
	}
	logrus.Debug("error in reading response of get net id map")
	return result, errors.New("Unknown command")
}

func (c *riakConn) Shutdown() error {
	if c.Client == nil {
		return errors.New("riak client is not connected")
	}
	return c.Client.Stop()
}
