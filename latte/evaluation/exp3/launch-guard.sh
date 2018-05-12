source ./env.sh
#!/bin/bash

SAFE_ADDR=http://$ADDR:$PORT
IAAS=152.3.145.38:444
IaaS=152.3.145.38:444

source ./functions
source ./manual-functions
source ./utils.sh

# In this case, we launch M containers on N VMs. Each container has
# several configuration parameters:
#   pidns, netns, mountns, utsns, ipcns, apparmor-profile, privileged
#
# the source used has following endorsements that have been loaded by authorizer
#   kernelBuildConfig(Source, "apparmor", 1)
#   kernelBuildConfig(Source, "selinux", 1)
#   kernelBuildConfig(Source, "seccomp", 1)
#   kernelBuildConfig(Source, "namespace", 1)
#   kernelBuildConfig(Source, "portext", 1)
#
# VMs are launched with the same image, which is built by building service,
#   the authorizer needs to point to default VM trust hub
#
# Containers are launched with the same image, which is built by building service
#   the authorizer needs to point to default container trust hub
#
# VM source is endorsed with attester property
#
# Verify that adding "mount" option fails it, and changing other config options fails it.


restartall


create() {
# configs
N=200
L=3
BUILDER="128.105.104.122:1-65535"

postVMInstance $IAAS "vm-builder" "image-builder" "128.105.104.122:1-65535" "192.168.1.0/24" "vpc-builder" "noauth:vm"
postEndorsementLink "noauth:vm" "vm-builder" "image-vm"
postEndorsementLink "noauth:docker" "vm-builder" "image-ctn"
postEndorsement "vm-builder" "image-vm" "source" "https://github.com/jerryz920/boot2docker.git#dev"
postEndorsement "vm-builder" "image-ctn" "source" "https://github.com/apache/spark.git#dev"
postVpcConfig1 $IAAS "vpc1" "launchGuard" "JustForTest"
  for n in `seq 1 $N`; do
    echo "posting instance $n"
    postVMInstance $IAAS "vm$n" "image-vm" "192.168.0.$n:1-65535" "192.168.$n.0/24" "vpc1" "noauth:vm"
    postInstanceConfig4 $IaaS "vm$n" "c1" "v1" "c2" "v2" "c3" "v3" "c4" "v4"
  done
}
create
LOG=${1:-launch-guard-log}
for n in `seq 1 100`; do
measureCheckClusterGuard anyone vm1 >> $LOG.cached
done

#restartproxy
#for n in `seq 1 20`; do
#measureCheckClusterGuard anyone vm1 >> $LOG.wo-netcache
#restartproxy
#done
