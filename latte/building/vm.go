package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type VmBuildParam struct {
	name   string
	repo   string
	source string
}

func (b *Builder) buildVM(param *VmBuildParam, w http.ResponseWriter) {

	os.Chdir(param.repo)
	imageName := fmt.Sprintf("vm-%s", param.name)
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	if log, err := cmd.CombinedOutput(); err != nil {
		logrus.Error("error in build ", err)
		logrus.Error("Logs: \n", string(log))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cmd = exec.Command("docker", "run", "-it", "--rm", imageName)
	image, err := cmd.Output()
	if err != nil {
		logrus.Error("error in run build ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := sha1.New()
	value := hash.Sum(image)
	b.postImage(hex.EncodeToString(value), param.source)
	w.Write(image)
}

func (b *Builder) BuildVM(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Error("error reading data: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dir, err := ioutil.TempDir("", "vm")
	defer os.RemoveAll(dir)
	if err != nil {
		logrus.Error("error creating temp dir: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		logrus.Error("error get working dir: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	if err = os.Chdir(dir); err != nil {
		logrus.Error("error switching dir: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.Chdir(wd)

	if err := Git(string(data)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	param := &VmBuildParam{
		name:   vars["name"],
		repo:   dir,
		source: string(data),
	}

	b.buildVM(param, w)
}
