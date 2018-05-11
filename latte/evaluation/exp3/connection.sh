source ./env.sh
#!/bin/bash

SAFE_ADDR=http://$ADDR:$PORT
IAAS=152.3.145.38:444
IaaS=152.3.145.38:444

source ./functions
source ./manual-functions
source utils.sh


# configs
N=100
L=3
BUILDER="128.105.104.122:1-65535"


create() {
# endorse the source from "simulated instance" to simplify the test
postVMInstance $IAAS "vm-builder" "image-builder" "128.105.104.122:1-65535" "192.168.1.0/24" "vpc-builder" "noauth:vm"
#postLinkImageOwner $IaaS "$BUILDER" "image-vm"
#postEndorsement "$BUILDER" "image-vm" "source" "https://github.com/jerryz920/boot2docker"
#postEndorsement "$BUILDER" "image-ctn" "source" "https://github.com/apache/spark"
#postEndorsement "$BUILDER" "image-spark" "source" "https://github.com/intel/hibench"

postEndorsementLink "noauth:vm" "vm-builder" "image-vm"
postEndorsementLink "noauth:docker" "vm-builder" "image-ctn"
postEndorsementLink "noauth:spark" "noauth:analytic" "image-spark"
postEndorsement "vm-builder" "image-vm" "source" "https://github.com/jerryz920/boot2docker.git#dev"
  n=1
  postVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
  postInstanceConfig4 $IaaS "vm$n" "imageRepo" "ipv4\\\"192.168.0.1\\\"" "c2" "v2" "c3" "v3" "c4" "v4"
  for n in `seq 2 $N`; do
    echo "posting instance $n"
    postVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
    postInstanceConfig4 $IaaS "vm$n" "imageRepo" "ipv4\\\"10.0.0.$n\\\"" "c2" "v2" "c3" "v3" "c4" "v4"
    #  postInstanceControl $IAAS $IAAS "vm$n"
  done
}

time create

LOG=${1:-connection-log}
for n in `seq 1 100`; do
measureCheckTrustedConnections "noauth:checkconn" vm1 >> $LOG
done
restartall
create
for n in `seq 1 100`; do
measureCheckTrustedConnections "noauth:checkconn" vm1 >> $LOG
restartproxy
done


