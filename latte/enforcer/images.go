package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ImageSource = "source"
)

type Image struct {
	Files    []string
	Property []string
}

type ImageIndex struct {
	Name     string
	Property []string
}

type ImageIndexes struct {
	Images []ImageIndex
}

func (enforcer *Enforcer) LoadImage(name string) error {
	enforcer.imageLock.Lock()
	defer enforcer.imageLock.Unlock()
	if _, ok := enforcer.images[name]; !ok {
		files := enforcer.imageStore.GetValues(name)
		if len(files) == 0 {
			return fmt.Errorf("no files in image %s", name)
		}
		propKey := ImagePropertyKey(name)
		properties := enforcer.imageStore.GetValues(propKey)
		enforcer.images[name] = &Image{
			Files:    files,
			Property: properties,
		}
	}
	return nil
}

func (enforcer *Enforcer) _putImage(name string) error {
	if i, ok := enforcer.images[name]; !ok {
		enforcer.imageStore.PutValues(name, i.Files)
		propKey := ImagePropertyKey(name)
		enforcer.imageStore.PutValues(propKey, i.Property)
	}
	return nil
}

func (enforcer *Enforcer) PutImage(name string) error {
	enforcer.imageLock.Lock()
	defer enforcer.imageLock.Unlock()
	return enforcer._putImage(name)
}

func ImagePropertyKey(name string) string {
	return name + "@property"
}

func (enforcer *Enforcer) indexImagePath(hash string) string {
	l := len(enforcer.imageServerAddr)
	if enforcer.imageServerAddr[l-1] == '/' {
		return fmt.Sprintf("http://%sget_index/%s", enforcer.imageServerAddr, hash)
	} else {
		return fmt.Sprintf("http://%s/get_index/%s", enforcer.imageServerAddr, hash)
	}
}

func (enforcer *Enforcer) imageContentPath(image string) string {
	l := len(enforcer.imageServerAddr)
	if enforcer.imageServerAddr[l-1] == '/' {
		return fmt.Sprintf("http://%sget_content/%s", enforcer.imageServerAddr, image)
	} else {
		return fmt.Sprintf("http://%s/get_content/%s", enforcer.imageServerAddr, image)
	}
}

func (enforcer *Enforcer) reportBackgroundTaskError(id int, image string, err error) {
	logrus.Errorf("bg task %d to download image %s failed with err: %s", id, image, err)
}

func (enforcer *Enforcer) downloadImage(name string) (*Image, error) {

	if i, ok := enforcer.GetCachedImage(name); ok {
		return i, nil
	}

	var image Image
	resp, err := http.Get(enforcer.imageContentPath(name))
	if err != nil {
		logrus.Debugf("error requesting for image %s: %s", name, err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("status code not OK: ", resp.StatusCode)
		logrus.Debugf("downloading image status error %s: %s", name, resp.StatusCode)
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&image); err != nil {
		logrus.Debugf("downloading image error in decoding: ", err)
		return nil, err
	}
	// do we actually need the lock? Whatever, does not matter too much, tens of nano second
	// if no contention
	enforcer.imageLock.Lock()
	enforcer.images[name] = &image
	enforcer._putImage(name)
	enforcer.imageLock.Unlock()
	return &image, nil
}

func (enforcer *Enforcer) downloadImagesBackground(indexes *ImageIndexes) {
	go func() {
		result := make([]*Image, len(indexes.Images))
		finished := make([]uint32, len(indexes.Images))
		for i := 0; i < len(finished); i++ {
			finished[i] = 0
		}

		for i, image := range indexes.Images {
			/// download things into result
			go func(slot int, index ImageIndex) {
				// properties should remain the same
				result[slot] = &Image{}

				resp, err := http.Get(enforcer.imageContentPath(index.Name))
				if err != nil {
					enforcer.reportBackgroundTaskError(slot, index.Name, err)
					return
				}
				if resp.StatusCode != http.StatusOK {
					err := fmt.Errorf("status code not OK: ", resp.StatusCode)
					enforcer.reportBackgroundTaskError(slot, index.Name, err)
					return
				}

				decoder := json.NewDecoder(resp.Body)
				if err := decoder.Decode(result[slot]); err != nil {
					enforcer.reportBackgroundTaskError(slot, index.Name, err)
					return
				}
				atomic.StoreUint32(&finished[slot], 1)
			}(i, image)
		}

		time.Sleep(time.Duration(enforcer.backgroundWaitSec) * time.Second)

		/// get finished
		enforcer.imageLock.Lock()
		for i := 0; i < len(finished); i++ {
			v := atomic.LoadUint32(&finished[i])
			if v == 1 {
				/// store must synced already
				name := indexes.Images[i].Name
				enforcer.images[name] = result[i]
				enforcer._putImage(name)
			}
		}
		enforcer.imageLock.Unlock()
	}()
}

func (enforcer *Enforcer) GetCachedImage(name string) (*Image, bool) {
	enforcer.imageLock.Lock()
	defer enforcer.imageLock.Unlock()
	if image, ok := enforcer.images[name]; ok {
		return image, true
	}
	return nil, false
}

func (enforcer *Enforcer) LookupCachedImage(hash string) (*ImageIndex, bool) {
	/// First check if local cache presents
	images := enforcer.imageStore.GetKey(hash)
	if len(images) > 0 {
		name := images[0]
		enforcer.imageLock.Unlock()
		defer enforcer.imageLock.Unlock()

		if image, ok := enforcer.images[name]; ok {
			return &ImageIndex{name, image.Property}, true
		}
		/// check if we have it in store
		propName := ImagePropertyKey(name)
		properties := enforcer.imageStore.GetValues(propName)
		if len(properties) > 0 {
			return &ImageIndex{name, properties}, true
		}
	}
	return nil, false
}

func (enforcer *Enforcer) LookupImageAndCache(hash string) (*ImageIndex, error) {
	if result, ok := enforcer.LookupCachedImage(hash); ok {
		return result, nil
	}

	/// slow path check http
	resp, err := http.Get(enforcer.indexImagePath(hash))
	if err != nil {
		logrus.Error("error requesting for index:", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logrus.Error("error reading response body. Response code: ", resp.StatusCode)

			return nil, err
		}
		err = fmt.Errorf("download error, code %d, content %s", resp.StatusCode, string(data))
		return nil, err
	}

	var index ImageIndexes
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&index); err != nil {
		remain, err := ioutil.ReadAll(decoder.Buffered())
		if err != nil {
			logrus.Errorf("error reading the decoder buffer for parsing index: ", err)
			return nil, err
		}
		logrus.Errorf("error parsing index body: %s, remaining %s", err, string(remain))
		return nil, err
	}

	// download the cache
	enforcer.downloadImagesBackground(&index)

	/// we assume the image is always the first
	if len(index.Images) == 0 {
		errors.New("Image not found")
		return nil, err
	}
	return &index.Images[0], err
}

func (enforcer *Enforcer) PolicyEnforced(properties []string, isbuild bool, policy *Policy) bool {
	for _, prop := range properties {
		parts := strings.Split(prop, "=")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == ImageSource {
			for _, w := range policy.Whitelist {
				if w == parts[1] {
					return true
				}
			}
		}
	}
	return false
}

func (enforcer *Enforcer) Eurosys18Enforce(hash string, pid int, proc *Proc, depth int) bool {

	if depth >= 4 {
		logrus.Error("recursive too deep, check!")
		return false
	}

	t1 := time.Now()
	image, err := enforcer.downloadImage(proc.Image)
	logrus.Printf("downloading time: %f\n", time.Since(t1).Seconds())

	if err != nil {
		idx := strings.Index(proc.Cmd, " ")
		if idx == -1 {
			logrus.Errorf("ENOINDEX %s %d %s #download, %s", proc.Cmd, pid, hash, err)
		} else {
			logrus.Errorf("ENOINDEX %s %d %s #download, %s", proc.Cmd[:idx], pid, hash, err)
		}
		return false
	}

	found := false

	for _, f := range image.Files {
		if f == hash {
			found = true
			break
		}
	}

	if !found {
		if proc.Image != SystemImage {
			ppid := Getppid(pid)
			if ppid == -1 {
				logrus.Errorf("proc %d parent %d missing", pid, ppid)
				fakeP := Proc{Image: SystemImage}
				return enforcer.Eurosys18Enforce(hash, -1, &fakeP, depth+1)
			}
			pproc := enforcer.GetProc(ppid)
			if pproc == nil {
				logrus.Errorf("proc %d parent %d missing", pid, ppid)
				fakeP := Proc{Image: SystemImage}
				return enforcer.Eurosys18Enforce(hash, ppid, &fakeP, depth+1)
			} else {
				return enforcer.Eurosys18Enforce(hash, ppid, pproc, depth+1)
			}
		}
	}
	return found
}
