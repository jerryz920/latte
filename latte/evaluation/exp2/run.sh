
	#flag.Var(&Addresses, "addr", "metadata service")
	#flag.IntVar(&Nthread, "nthread", 1, "num of cocurrent thread")
	#flag.IntVar(&NumVM, "nvm", 200, "num of vm instance")
	#flag.IntVar(&NumLevel, "nlevel", 3, "num of level")
	#flag.BoolVar(&RandAccess, "rand", false, "random access")
	#flag.IntVar(&Round, "round", 1, "random access")

msdir="/openstack/go/src/github.com/jerryz920/conferences/latte/proxy"
riakdir="/openstack/safe/uber-safe/cluster-scripts"
LOG=glog
workdir=`pwd`
mkdir results -p

ulimit -n 90000

config() {
  args=""
  for n in 1 2 3 4; do
    args="$args --addr http://localhost:$((19851+n))"
  done

  # 1 = nthread
    args="$args --nthread $1"
  # 2 = nvm
    args="$args --nvm $2"
  # 3 = member
    args="$args --member $3"
  # 4 = round
    args="$args --round $4"
   echo "$args"

}

export myid=10
run() {
  echo "starting $* exps"
  echo "restarting riak"
  cd $riakdir
  bash all-restart.sh
  sleep 5
  echo "starting metadata service"
  cd $workdir; bash stop.sh 4; bash start.sh 4
  sleep 20
  args=`config $*`
  cd $workdir
  echo "running ./exp2 $args"
  ./exp2 $args 2>> $LOG 
  mv $LOG results/$LOG-$myid
  myid=$((myid+1))
}

#for n in 4; do
#  for j in 1; do
##    run 4 $n 3 1 $j 
#    run 1 $n 3 0 $j 
#  done
#done
#
## vary thread
#for n in 2; do
#  for j in 1; do
##    run $n 4 3 1 $j 
#    run $n 4 3 0 $j 
#  done
#done
#
## vary level
#for n in 1; do
#  for j in 1; do
##    run 4 4 $n 1 $j 
#    run 4 4 $n 0 $j 
#  done
#done
# vary nvm
#for n in 128; do
#  for j in 1; do
#    run 64 $n 3 1 $j
#    #run 64 $n 3 0 $j
#  done
#done

# vary level
#for n in 1; do
#  for j in 1; do
#    run 4 1024 $n 1 $j
#    run 4 1024 $n 0 $j
#  done
#done

# vary num member 
for n in 128; do
  for m in 20 30 40 50; do
  for j in 1 2 3 4 5; do
    run 128 10240 $m $j 
  done
done
done
