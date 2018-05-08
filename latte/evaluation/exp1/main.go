package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	base "github.com/jerryz920/conferences/latte/evaluation"
)

type MetadataServices []string

var (
	Addresses  MetadataServices
	Nthread    int
	NumVM      int
	NumLevel   int
	RandAccess int
	Round      int
	IaaS       string = "152.3.145.38:444"
)

func (m *MetadataServices) String() string {
	return fmt.Sprintf("%v", *m)
}

func (m *MetadataServices) Set(val string) error {
	*m = append(*m, val)
	return nil
}

var (
	levels = []int{200, 50, 1}
)

func config() {
	flag.Var(&Addresses, "addr", "metadata service")
	flag.IntVar(&Nthread, "nthread", 1, "num of cocurrent thread")
	flag.IntVar(&NumVM, "nvm", 200, "num of vm instance")
	flag.IntVar(&NumLevel, "nlevel", 3, "num of level")
	flag.IntVar(&RandAccess, "rand", 0, "random access")
	flag.IntVar(&Round, "round", 1, "random access")
	flag.Parse()
}

func Upfront(client *base.MetadataClient) {
	err := client.Request(0, "/postVMInstance", IaaS,
		"vm-builder",
		"image-builder",
		"128.105.104.122:1-65535",
		"172.16.0.0/24",
		"vpc-builder",
		"noauth:vm")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	err = client.Request(0, "/postEndorsementLink", "noauth:vm",
		"vm-builder",
		"image-vm")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	err = client.Request(0, "/postEndorsementLink", "noauth:docker",
		"vm-builder",
		"image-ctn")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	err = client.Request(0, "/postEndorsementLink", "noauth:spark",
		"noauth:analytic",
		"image-spark")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	err = client.Request(0, "/postEndorsement", "vm-builder", "image-vm", "source",
		"https://github.com/jerryz920/boot2docker.git#dev")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	err = client.Request(0, "/postEndorsement", "vm-builder", "image-ctn", "source",
		"https://github.com/apache/spark.git#dev")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	err = client.Request(0, "/postEndorsement", "noauth:analytic", "image-spark", "source",
		"https://github.com/intel/hibench.git#dev")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}

func main() {

	config()
	if NumVM%Nthread != 0 {
		logrus.Fatalf("NumVM % nthread must = 0, %d %d", NumVM, Nthread)
		os.Exit(1)
	}
	if NumLevel > 3 {
		logrus.Fatal("Mostly 3 levels, ", NumLevel)
		os.Exit(1)
	}
	clients := make([]*base.MetadataClient, len(Addresses))
	chans := make([]chan bool, Nthread)
	i := 0

	for _, addr := range Addresses {
		clients[i] = base.NewClient(addr)
		i++
	}
	formatter := logrus.JSONFormatter{}
	formatter.DisableTimestamp = true
	logrus.SetFormatter(&formatter)
	logrus.SetLevel(logrus.InfoLevel)

	t1 := time.Now()

	Upfront(clients[0])

	for i = 0; i < Nthread; i++ {
		chans[i] = make(chan bool)
		if RandAccess == 1 {
			go WorkRandom(clients[i%len(clients)], i, chans[i])
		} else {
			go Work(clients[i%len(clients)], i, chans[i])
		}
	}

	for i = 0; i < Nthread; i++ {
		<-chans[i]
	}

	t2 := time.Now().Sub(t1).Seconds()
	logrus.WithField("nvm", &NumVM).WithField("rand", &RandAccess).
		WithField("nthread", &Nthread).WithField("nlevel", &NumLevel).
		WithField("time", &t2).Info("finish")

}
