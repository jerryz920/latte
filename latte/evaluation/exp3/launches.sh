source ./env.sh
#!/bin/bash

SAFE_ADDR=http://$ADDR:$PORT
IAAS=152.3.145.38:444
IaaS=152.3.145.38:444

source ./functions
source ./manual-functions
source ./utils.sh


# configs
N=10
L=3

restartall
create() {
BUILDER="128.105.104.122:1-65535"

# endorse the source from "simulated instance" to simplify the test
postVMInstance $IAAS "vm-builder" "image-builder" "128.105.104.122:1-65535" "192.168.1.0/24" "vpc-builder" "noauth:vm"
#postLinkImageOwner $IaaS "$BUILDER" "image-vm"
#postEndorsement "$BUILDER" "image-vm" "source" "https://github.com/jerryz920/boot2docker"
#postEndorsement "$BUILDER" "image-ctn" "source" "https://github.com/apache/spark"
#postEndorsement "$BUILDER" "image-spark" "source" "https://github.com/intel/hibench"

postEndorsementLink "noauth:vm" "vm-builder" "image-vm"
postEndorsementLink "noauth:vm" "vm-builder" "image-vm"
postEndorsementLink "noauth:docker" "vm-builder" "image-ctn"
postEndorsementLink "noauth:spark" "noauth:analytic" "image-spark"
postEndorsement "vm-builder" "image-vm" "source" "https://github.com/jerryz920/boot2docker.git#dev"
postEndorsement "vm-builder" "image-ctn" "source" "https://github.com/apache/spark.git#dev"
postEndorsement "noauth:analytic" "image-spark" "source" "https://github.com/intel/hibench.git#dev"
  for n in `seq 1 $N`; do
    echo "posting instance $n"
    postVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
    postInstanceConfig4 $IaaS "vm$n" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4"
    #  postInstanceControl $IAAS $IAAS "vm$n"
    if [ $L -le 1 ]; then
      continue;
    fi
    for m in `seq 1 10`; do
      postInstance "192.168.0.$n:1-65535" "vm$n-ctn$m" "image-ctn" "192.168.$n.$m:1-65535" "noauth:docker"
      postInstanceConfig5 "192.168.0.$n:1-65535" "vm$n-ctn$m" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4" "c5" "v5"
      #    postInstanceControl $IAAS "vm$n" "vm$n-ctn$m"
      if [ $L -le 2 ]; then
	continue;
      fi
      for l in `seq 1 2`; do
	port1="3${l}000"
	port2="3${l}999"
	postInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "image-spark" "192.168.$n.$m:$port1-$port2"  "noauth:spark"
	postInstanceConfig5 "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4" "c5" "v5"
	#      postInstanceControl $IAAS "vm$n-ctn$m" "vm$n-spark$l"
      done
    done
  done
}

create

LOG=${1:-launches-log}
for n in `seq 1 100`; do
measureCheckLaunches anyone vm1-ctn1 image-ctn >> $LOG.cached
done

#restartsafe
#
#for n in `seq 1 20`; do
#measureCheckLaunches anyone vm1-ctn1 image-ctn >> $LOG.wo-objcache
#restartsafe
#done

#restartproxy
#
#for n in `seq 1 20`; do
#measureCheckLaunches anyone vm1-ctn1 image-ctn >> $LOG.wo-netcache
#restartproxy
#done

#restartfe
#for n in `seq 1 20`; do
#measureCheckLaunches anyone vm1-ctn1 image-ctn >> $LOG.wo-cache
#restartfe
#done
