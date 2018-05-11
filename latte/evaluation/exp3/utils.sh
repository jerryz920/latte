source ./env.sh

msdir="/openstack/go/src/github.com/jerryz920/conferences/latte/proxy"
riakdir="/openstack/safe/uber-safe/cluster-scripts"
workdir=`pwd`

restartall() {
#  cd $riakdir
#  bash all-restart.sh
#  sleep 4
#  echo "starting metadata service"
#  cd $workdir; bash stop.sh 1; bash start.sh 1
#  sleep 20
echo
}

restartproxy(){
#  killall metaserver
#  ./metaserver --debug --addr 10.10.1.3:8087 --addr 10.10.1.4:8087 --addr 10.10.1.5:8087 --safe localhost:7777 --listen 0.0.0.0:19852  >>perflog-meta-1 2>&1 &
#  sleep 2
echo
}

