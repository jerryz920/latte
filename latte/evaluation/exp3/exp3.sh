source ./env.sh
#!/bin/bash


SAFE_ADDR=http://$ADDR:$PORT
IAAS=152.3.145.38:444
IaaS=152.3.145.38:444

source ./functions
source ./manual-functions
source ./utils.sh

do_cache() {
  for n in `seq 1 $1`; do
    measurePostVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
  done
  n=1
  for m in `seq 1 $1`; do
    measurePostInstance "192.168.0.$n:1-65535" "vm$n-ctn$m" "image-ctn" "192.168.$n.$m:1-65535" "noauth:docker"
  done
  m=1
  for l in `seq 1 $1`; do
    port1=`expr 30000 + $l \* 10`
    port2=`expr 30000 + $l \* 10 + 9`
    measurePostInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "image-spark" "192.168.$n.$m:$port1-$port2"  "noauth:spark"
  done
  restartproxy
  for n in `seq 1 $1`; do
    measureCheckFetch checking  192.168.0.1:1000
  done
  n=1
  for m in `seq 1 $1`; do
    measureCheckFetch checking  192.168.1.1:1000
  done
  m=1
  for l in `seq 1 $1`; do
    measureCheckFetch checking  192.168.1.1:30010
  done
}

no_cache() {
  for n in `seq 1 $1`; do
    measurePostVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
    restartproxy
  done
  n=1
  for m in `seq 1 $1`; do
    measurePostInstance "192.168.0.$n:1-65535" "vm$n-ctn$m" "image-ctn" "192.168.$n.$m:1-65535" "noauth:docker"
    restartproxy
  done
  m=1
  for l in `seq 1 $1`; do
    port1=`expr 30000 + $l \* 10`
    port2=`expr 30000 + $l \* 10 + 9`
    measurePostInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "image-spark" "192.168.$n.$m:$port1-$port2"  "noauth:spark"
    restartproxy
  done
  restartproxy
  for n in `seq 1 $1`; do
    measureCheckFetch checking  192.168.0.1:1000
    restartproxy
  done
  n=1
  for m in `seq 1 $1`; do
    measureCheckFetch checking  192.168.1.1:1000
    restartproxy
  done
  m=1
  for l in `seq 1 $1`; do
    measureCheckFetch checking  192.168.1.1:30010
    restartproxy
  done
}


mkdir -p results

# we measure 100 100 100 
for run in 1 2 3 4 5; do
  restartall
  do_cache 20 >> cache-create.log
  restartall
  no_cache 20 >> nocache-create.log
done
mv cache-create.log results
mv nocache-create.log results


