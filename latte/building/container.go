package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type containerBuildParam struct {
	name   string
	repo   string
	source string
}

func (b *Builder) buildContainer(param *containerBuildParam, w http.ResponseWriter) {

	os.Chdir(param.repo)
	imageName := fmt.Sprintf("container-%s", param.name)
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	if log, err := cmd.CombinedOutput(); err != nil {
		logrus.Error("error in build ", err)
		logrus.Error("Logs: \n", string(log))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cmd = exec.Command("/usr/bin/upload_image.sh", imageName)
	log, err := cmd.Output()
	if err != nil {
		logrus.Error("error in uploading image hash", err)
		logrus.Error("Logs: ", log)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadFile("/tmp/docker/image.tar")
	if err != nil {
		logrus.Error("error in posting image property", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b.postImage(imageName, param.source)
	w.Write(data)
}

func (b *Builder) BuildContainer(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Error("error reading data: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dir, err := ioutil.TempDir("", "container")
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

	param := &containerBuildParam{
		name:   vars["name"],
		repo:   dir,
		source: string(data),
	}

	b.buildContainer(param, w)
}
