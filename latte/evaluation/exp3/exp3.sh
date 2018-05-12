source ./env.sh
#!/bin/bash


SAFE_ADDR=http://$ADDR:$PORT
IAAS=152.3.145.38:444
IaaS=152.3.145.38:444

source ./functions
source ./manual-functions
source ./utils.sh


measure() {
  for n in `seq 1 100`; do
    postVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
  done
  n=1
  for m in `seq 1 100`; do
    postInstance "192.168.0.$n:1-65535" "vm$n-ctn$m" "image-ctn" "192.168.$n.$m:1-65535" "noauth:docker"
  done
  m=1
  for l in `seq 1 100`; do
    port1=`expr 30000 + $l \* 10`
    port2=`expr 30000 + $l \* 10 + 9`
    postInstance "192.168.$n.$m:1-65535" "vm$n-ctn$m-spark$l" "image-spark" "192.168.$n.$m:$port1-$port2"  "noauth:spark"
  done
  #echo "CACHE"
  #for n in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.0.1:1000 >> $2.cached
  #done
  #n=1
  #for m in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.1.1:1000 >> $2.cached
  #done
  #m=1
  #for l in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.1.1:30010 >> $2.cached
  #done
  #echo "PROXY"
  #restartproxy
  #for n in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.0.1:1000 >> $2.wo-netcache
  #  restartproxy
  #done
  #n=1
  #for m in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.1.1:1000 >> $2.wo-netcache
  #  restartproxy
  #done
  #m=1
  #for l in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.1.1:30010 >> $2.wo-netcache
  #  restartproxy
  #done
  #echo "SAFE"
  #restartsafe
  #for n in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.0.1:1000 >> $2.wo-objcache
  #  restartsafe
  #done
  #n=1
  #for m in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.1.1:1000 >> $2.wo-objcache
  #  restartsafe
  #done
  #m=1
  #for l in `seq 1 $1`; do
  #  measureCheckFetch checking  192.168.1.1:30010 >> $2.wo-objcache
  #  restartsafe
  #done

  echo "FE"
  restartfe
  for n in `seq 1 $1`; do
    measureCheckFetch checking  192.168.0.1:1000 >> $2.wo-cache
    restartfe
  done
  n=1
  for m in `seq 1 $1`; do
    measureCheckFetch checking  192.168.1.1:1000 >> $2.wo-cache
    restartfe
  done
  m=1
  for l in `seq 1 $1`; do
    measureCheckFetch checking  192.168.1.1:30010 >> $2.wo-cache
    restartfe
  done

}


mkdir -p results

# we measure 100 100 100 
for run in 1; do
  restartall
  measure 20 $1
done


