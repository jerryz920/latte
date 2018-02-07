package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func Getppid(proc int) int {
	f, err := os.Open(fmt.Sprintf("/proc/%d/stat"))
	if err != nil {
		logrus.Errorf("can not open process %d, %s", proc, err)
		return -1
	}
	defer f.Close()

	var s string
	var pid int
	fmt.Fscan(f, &s)
	fmt.Fscan(f, &s)
	fmt.Fscan(f, &s)
	fmt.Fscan(f, &pid)
	return pid
}

func Getexec(proc int) (string, string) {
	fpath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", proc))
	if err != nil {
		logrus.Errorf("can not open process %d, %s", proc, err)
		return "", ""
	}
	rootDir := fmt.Sprintf("/proc/%d/root", proc)
	f, err := os.Open(filepath.Join(rootDir, fpath))
	if err != nil {
		return "", ""
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		logrus.Error("can not compute hash of exec ", fpath)
		return "", ""
	}
	return fpath, hex.EncodeToString(h.Sum(nil))
}

func Gethash(proc int, fpath string) string {
	rootDir := fmt.Sprintf("/proc/%d/root", proc)
	f, err := os.Open(filepath.Join(rootDir, fpath))
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		logrus.Error("can not compute hash of exec ", fpath)
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}
