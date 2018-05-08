package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	logrus "github.com/sirupsen/logrus"
)

func (inst *CachedInstance) Copy() *CachedInstance {
	return &CachedInstance{
		Lport: inst.Lport,
		Rport: inst.Rport,
		Ip:    inst.Ip,
		ID: &InstanceCred{
			Pid:  inst.ID.Pid,
			PPid: inst.ID.PPid,
			Cidr: inst.ID.Cidr,
			Type: inst.ID.Type,
		},
	}
}

type Cache interface {
	/// It's safe to modify the cached contents. It is fast anyway so we do deep copy
	GetInstanceFromNetMap(ip net.IP, port int) (CachedInstance, int)
	GetInstanceFromID(pid string) (CachedInstance, int)
	PutInstance(inst *CachedInstance) int
	DelInstance(ip net.IP, lport, rport int, pid string) int
	SetLog(*logrus.Logger)
}

type cacheImpl struct {
	sync.Mutex
	conn RiakConn
	pmap *Pmap
	id   int
	log  *logrus.Logger
}

var (
	id int = 0
)

func NewCache(c RiakConn) Cache {
	id++
	return &cacheImpl{
		conn: c,
		pmap: NewPmap(),
		id:   id,
	}
}

func (c *cacheImpl) SetLog(l *logrus.Logger) {
	c.log = l
}

func (c *cacheImpl) reloadCache(ip net.IP) int {
	t1 := time.Now()
	pmaps, err := c.conn.GetAllNetID(ip)
	if err != nil {
		c.log.Debugf("%d reloading cache for %s: %s", c.id, ip, err)
		return http.StatusInternalServerError
	}
	for _, inst := range pmaps {
		// Creating same Index is not a problem, since the
		// key field in interval tree will always be the same, thus
		// it won't yield multiple copies.
		tmp := inst
		c.pmap.PutCachedInstance(&tmp)
	}
	c.log.Infof("%d PERFCACHE RELOAD %f", c.id, time.Now().Sub(t1).Seconds())
	return http.StatusOK

}

func (c *cacheImpl) reloadUUID(uuid string) (*CachedInstance, int) {
	// search for this uuid, if found, then reload the cache
	t1 := time.Now()
	pmap, err := c.conn.SearchIDNet(uuid)
	/// In theory there should be one
	if err != nil {
		c.log.Debug("searchIDNet failure ", err)
		return nil, http.StatusInternalServerError
	}

	if pmap == nil || len(pmap) == 0 {
		c.log.Debug("searchIDNet, not found")
		return nil, http.StatusNotFound
	}

	if len(pmap) > 1 {
		logrus.Warning("can not have more than one instance with same UUID: %s, %s",
			pmap[0].ID.Pid, pmap[0].ID.PPid)
	}

	/// When an instance is hot, bring up the entire VM
	if c.pmap.Loaded(pmap[0].Ip.String()) {
		logrus.Warningf("Inconsistency in cache state, the IP %s is already loaded but UUID %s not found in cache.",
			pmap[0].Ip, uuid)
	}
	c.log.Info("PERFCACHE RELOAD-ID ", time.Now().Sub(t1).Seconds())

	/// Same design with load cache using IP.
	return &pmap[0], c.reloadCache(pmap[0].Ip)

}

func (c *cacheImpl) GetInstanceFromNetMap(ip net.IP, port int) (CachedInstance, int) {
	t1 := time.Now()
	c.Lock()
	defer c.Unlock()
	ipstr := ip.String()

	index, err := c.pmap.GetIndex(ipstr, port)
	if err != nil {
		c.log.Debug("GetInstanceFromNetMap, getting index: ", err)
		return CachedInstance{}, http.StatusInternalServerError
	}
	if index == nil {
		/// reload the cache and see if we can find it
		if status := c.reloadCache(ip); status != http.StatusOK {
			c.log.Debug("err GetInstanceFromNetMap, visiting backend store")
			return CachedInstance{}, status
		}
		index, err = c.pmap.GetIndex(ipstr, port)

		if err != nil {
			c.log.Debug("err GetInstanceFromNetMap, after cached loaded: ", err)
			return CachedInstance{}, http.StatusInternalServerError
		}
		/// Not found
		if index == nil {
			c.log.Debug("Not found the instance")
			return CachedInstance{}, http.StatusNotFound
		}
	}

	result, _ := c.pmap.GetCachedInstance(index.P)
	c.log.Info("PERFCACHE GetInstanceFromNetMap ", time.Now().Sub(t1).Seconds())
	//// result should never be null!
	return *result.Copy(), http.StatusOK

}

func (c *cacheImpl) GetInstanceFromID(pid string) (CachedInstance, int) {
	t1 := time.Now()
	var status int
	c.Lock()
	defer c.Unlock()
	inst, err := c.pmap.GetCachedInstance(pid)
	if err != nil {
		c.log.Debugf("Looking up UUID %s from backend", pid)
		inst, status = c.reloadUUID(pid)
		if status != http.StatusOK {
			return CachedInstance{}, status
		}
	}
	/// result won't be nil, error will be returned for not found
	c.log.Info("PERFCACHE GetInstanceFromUUID ", time.Now().Sub(t1).Seconds())
	return *inst, http.StatusOK
}

/// Creation of an instance, store it to backend
/// Q: can we accelerate by async posting?
func (c *cacheImpl) PutInstance(inst *CachedInstance) int {
	t1 := time.Now()
	c.Lock()
	defer c.Unlock()

	// We are using write through strategy here. This slows
	// down creation, but won't affect read of the same
	// instance.
	c.pmap.PutCachedInstance(inst)
	if err := c.conn.PutNetIDMap(inst.Ip, inst.Lport, inst.Rport, inst.ID); err != nil {
		c.log.Debug("PutInstance error in PutNetIDMap: ", err)
		return http.StatusInternalServerError
	}
	c.log.Info("PERFCACHE PutInstance ", time.Now().Sub(t1).Seconds())
	return http.StatusOK
}

func (c *cacheImpl) DelInstance(ip net.IP, lport, rport int, pid string) int {
	t1 := time.Now()
	c.Lock()
	defer c.Unlock()
	c.pmap.DelCachedInstanceAlt(ip, lport, rport, pid)
	/// We may just delete in async way.
	if err := c.conn.DelNetIDMap(ip, lport, rport); err != nil {
		c.log.Debug("DelInstance error in DelNetIDMap: ", err)
		return http.StatusInternalServerError
	}
	c.log.Info("PERFCACHE DelInstance ", time.Now().Sub(t1).Seconds())
	return http.StatusOK
}
