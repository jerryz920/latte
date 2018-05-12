source ./env.sh

msdir="/openstack/go/src/github.com/jerryz920/conferences/latte/proxy"
riakdir="/openstack/safe/uber-safe/cluster-scripts"
workdir=`pwd`

restartall() {
  cd $riakdir
  bash all-restart.sh
  sleep 4
  echo "starting metadata service"
  cd $workdir; bash stop.sh 1; bash start.sh 1
  sleep 30
echo
}

restartsafe() {
  docker rm -f safe1
  bash start.sh 1 1>>debugsafe.out 2>&1
  sleep 20
}

restartproxy(){
  killall -KILL metaserver
  sleep 0.5
  bash start.sh 1 1>>debugsafe.out 2>&1
  sleep 1
echo
}

restartfe() {
  bash stop.sh 1;
  bash start.sh 1 1>>debugsafe.out 2>&1
  sleep 30
}
