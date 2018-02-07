package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type makefileBuildParam struct {
	name   string
	repo   string
	source string
}

func cweCount(data []byte) string {
	return "0"
}
func (b *Builder) swamp(dir string) string {
	return ""
}

func (b *Builder) buildMakefile(param *makefileBuildParam, w http.ResponseWriter) {

	os.Chdir(param.repo)
	cmd := exec.Command("make")
	if log, err := cmd.CombinedOutput(); err != nil {
		logrus.Error("error in build ", err)
		logrus.Error("Logs: \n", string(log))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadFile(param.name)
	if err != nil {
		logrus.Error("error in generating the name", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hash := sha1.New()
	id := hex.EncodeToString(hash.Sum(data))

	err = b.postImage(id, param.source)
	if err != nil {
		logrus.Error("error posintg image", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	xmlFile := b.swamp(param.repo)
	xmlData, err := ioutil.ReadFile(xmlFile)
	if err != nil {
		logrus.Error("error requesting swamp", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	b.postImage(id, cweCount(xmlData))

	w.Write(data)
}

func (b *Builder) BuildMakefile(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Error("error reading data: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dir, err := ioutil.TempDir("", "makefile")
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

	param := &makefileBuildParam{
		name:   vars["name"],
		repo:   dir,
		source: string(data),
	}

	b.buildMakefile(param, w)
}
