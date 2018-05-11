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
BUILDER="128.105.104.122:1-65535"


create() {
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
      for l in `seq 1 4`; do
	port1="3${l}000"
	port2="3${l}999"
	postInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "image-spark" "192.168.$n.$m:$port1-$port2"  "noauth:spark"
	postInstanceConfig5 "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4" "c5" "v5"
	#      postInstanceControl $IAAS "vm$n-ctn$m" "vm$n-spark$l"
      done
    done
  done

#export AUTH=1
postCluster vm1-ctn1 "spark" "ownerguardtext" "joinguardtext"
for n in `seq 2 8`; do
  for m in `seq 2 8`; do
    export AUTH=1
    postMembership vm1-ctn1 "spark" vm$n-ctn$m
    export AUTH=$n
    postAckMembership vm$n-ctn$m "spark" vm1-ctn1
  done
done
}

create

restartproxy
for n in `seq 1 100`; do
  measureCheckTrustedCluster "noauth:alice" vm2-ctn5 >> $LOG
done
restartall

create
for n in `seq 1 100`; do
  measureCheckTrustedCluster "noauth:alice" vm2-ctn5 >> $LOG
  restartproxy
done


#checkFetch haha vm1-ctn1-spark1
#checkAttester $IaaS $IaaS
#checkBuilder $IaaS vm-builder
#checkLaunches anyone vm1 image-vm 
#checkBuildsFrom anyone vm1 image-vm "https://github.com/jerryz920/boot2docker.git#dev"
#checkEndorse anyone vm1 image-vm attester 1
#checkEndorse anyone vm1 "https://github.com/jerryz920/boot2docker.git#dev" attester 1
#jcheckAttester anyone vm1
#checkAttester anyone vm1-ctn1
#checkAttester $IaaS vm1-ctn1
#






#postLinkImageOwner "$IaaS" "$BUILDER" "image-ctn"
#postLinkImageOwner "$IaaS" "$BUILDER" "image-spark"
#
#
## create source for image-vm, image-spark, image-ctn, make image-builder the builder image
#
#
#
remove() {
  for n in `seq 1 $N`; do
    echo "deleting instance $n"
    if [ $L -gt 1 ]; then
      for m in `seq 1 5`; do
	if [ $L -gt 2 ]; then
	  for l in `seq 1 5`; do
	    lazyDeleteInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" 
	    #delInstanceConfig5 "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4" "c5" "v5"
	    #delInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "image-spark" "192.168.$n.$m:32000-65535" 
	    echo -n
	  done
	  lazyDeleteInstance "vm$n" "vm$n-ctn$m" 
	  #delInstanceConfig5 "192.168.0.$n:1-65535" "vm$n-ctn$m" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4" "c5" "v5"
	  #delInstance "192.168.0.$n:1-65535" "vm$n-ctn$m" "image-ctn" "192.168.$n.$m:1-65535" 
	fi
      done
    fi
    lazyDeleteInstance $IAAS "vm$n" 
    #delInstanceConfig4 $IaaS "vm$n" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4"
    #delVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1"
    #delInstanceConfig4 $IaaS "vm$n" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4"
  done
}
