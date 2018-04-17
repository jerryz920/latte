package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"

	jhttp "github.com/jerryz920/utils/goutils/http"
	eurosys18 "github.com/jerryz920/utils/random/conferences/eurosys18"
	"github.com/sirupsen/logrus"
)

type Builder struct {
	addr          string /// Metadata service address
	vmLock        sync.Mutex
	containerLock sync.Mutex
	makefileLock  sync.Mutex
}

func (b *Builder) postImage(name string, source string) error {

	body := bytes.Buffer{}
	body.WriteString(
		fmt.Sprintf(`{"principal": "%s", "otherValues": ["%s", "", "%s"]}`,
			name, source))

	api := fmt.Sprintf("http://%s%s", b.addr, "/postImageProperty")
	resp, err := http.Post(api, "application/json", &body)
	if err != nil {
		logrus.Error("error in posting image: ", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logrus.Error("error read response body ", err)
			data = []byte{}
		}
		logrus.Debugf("server status %d, body, %s", resp.StatusCode, data)
		return fmt.Errorf("post image fail: status code: %s", resp.Status)
	}
	return nil
}

func Git(url string) error {
	cmd := exec.Command("git", "clone", url, ".")
	if combined, err := cmd.CombinedOutput(); err != nil {
		logrus.Error("clone error ", err)
		logrus.Error("log:\n", string(combined))
		return err
	}
	return nil
}

func ProbeDocker() {
	ccmd := exec.Command("dockerd", "-v")
	_, err := ccmd.CombinedOutput()
	if err != nil {
		logrus.Fatal("No docker daemon running!")
	}

	scmd := exec.Command("docker", "ps")
	_, err = scmd.CombinedOutput()
	if err != nil {
		logrus.Fatal("No docker client!")
	}

	tcmd := exec.Command("git")
	_, err = tcmd.CombinedOutput()
	if err != nil {
		logrus.Fatal("No tar found!")
	}

	if _, err := os.Stat("/usr/bin/upload_image.sh"); err != nil {
		logrus.Fatal("Container uploading script not available!", err)
	}
}

func main() {
	ProbeDocker()
	server := jhttp.NewAPIServer(nil)
	builder := Builder{}
	server.AddRoute("/vm/{name}", builder.BuildVM, "building VM")
	server.AddRoute("/container/{name}", builder.BuildContainer, "building container")
	server.AddRoute("/makefile/{name}", builder.BuildMakefile, "building makefile")

	server.ListenAndServe(eurosys18.BuildingServiceAddress)

}
