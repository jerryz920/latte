source ./env.sh
#!/bin/bash

SAFE_ADDR=http://$ADDR:$PORT
IAAS=152.3.145.38:444
IaaS=152.3.145.38:444

source ./functions
source ./manual-functions
source utils.sh


# configs
N=20
L=3
BUILDER="128.105.104.122:1-65535"


create() {
# endorse the source from "simulated instance" to simplify the test
postVMInstance $IAAS "vm-builder" "image-builder" "128.105.104.122:1-65535" "192.168.1.0/24" "vpc-builder" "noauth:vm"
postVMInstance $IAAS "vm-scanner" "image-scanner" "128.105.104.123:1-65535" "192.168.2.0/24" "vpc-scanner" "noauth:docker"
#postLinkImageOwner $IaaS "$BUILDER" "image-vm"
#postEndorsement "$BUILDER" "image-vm" "source" "https://github.com/jerryz920/boot2docker"
#postEndorsement "$BUILDER" "image-ctn" "source" "https://github.com/apache/spark"
#postEndorsement "$BUILDER" "image-spark" "source" "https://github.com/intel/hibench"

postEndorsementLink "noauth:vm" "vm-builder" "image-vm"
postEndorsementLink "noauth:docker" "vm-builder" "image-ctn"
postEndorsementLink "noauth:docker" "vm-scanner" "image-ctn"
postEndorsement "vm-builder" "image-vm" "source" "https://github.com/jerryz920/boot2docker.git#dev"
postEndorsement "vm-builder" "image-ctn" "source" "https://github.com/docker/ubuntu.git#xenial"
postEndorsement "vm-builder" "image-scanner" "source" "https://github.com/arminc/clair-scanner.git#master"

for n in `cat cve.out`; do
  m=`bash -c "echo $n"`
  postEndorsement "vm-scanner" "image-cnt" "cve" $m
done
  for n in `seq 1 $N`; do
    echo "posting instance $n"
    postVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
    postInstanceConfig4 $IaaS "vm$n" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4"
    #  postInstanceControl $IAAS $IAAS "vm$n"
    if [ $L -le 1 ]; then
      continue;
    fi
    for m in `seq 1 20`; do
      postInstance "192.168.0.$n:1-65535" "vm$n-ctn$m" "image-ctn" "192.168.$n.$m:1-65535" "noauth:docker"
      postInstanceConfig5 "192.168.0.$n:1-65535" "vm$n-ctn$m" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4" "c5" "v5"
    done
  done
}

create

LOG=${1:-quality-log}
for n in `seq 1 20`; do
measureCheckCodeQuality "noauth:codeworker" 192.168.1.2:3000 >> LOG
done
restartall
create

for n in `seq 1 20`; do
measureCheckCodeQuality "noauth:codeworker" 192.168.1.2:3000 >> LOG
restartproxy
done


