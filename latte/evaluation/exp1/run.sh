
	#flag.Var(&Addresses, "addr", "metadata service")
	#flag.IntVar(&Nthread, "nthread", 1, "num of cocurrent thread")
	#flag.IntVar(&NumVM, "nvm", 200, "num of vm instance")
	#flag.IntVar(&NumLevel, "nlevel", 3, "num of level")
	#flag.BoolVar(&RandAccess, "rand", false, "random access")
	#flag.IntVar(&Round, "round", 1, "random access")

msdir="/openstack/go/src/github.com/jerryz920/conferences/latte/proxy"
riakdir="/openstack/safe/uber-safe/cluster-scripts"
LOG=perf-log
workdir=`pwd`


config() {
  args=""
  for n in 1; do
    args="$args --addr http://compute4:$((19850+n))"
  done

  # 1 = nthread
    args="$args --nthread $1"
  # 2 = nvm
    args="$args --nvm $2"
  # 3 = nlevel
    args="$args --nlevel $3"
  # 4 = ordered
    args="$args --rand $4"
  # 5 = round
    args="$args --round $5"
   echo "$args"

}

run() {
#  echo "starting $* exps"
#  echo "restarting riak"
#  cd $riakdir
#  bash all-restart.sh
#  sleep 10
#  echo "starting metadata service"
#  ssh compute4 "cd $msdir; bash stop.sh 4; bash start.sh 4"
#  sleep 20
  args=`config $*`
  cd $workdir
  echo "running ./exp1 $args"
  ./exp1 $args >> $LOG 2>&1
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
for n in 128; do
  for j in 1; do
    run 64 $n 3 1 $j
    #run 64 $n 3 0 $j
  done
done

# vary level
#for n in 1; do
#  for j in 1; do
#    run 4 1024 $n 1 $j
#    run 4 1024 $n 0 $j
#  done
#done

# vary nvm
#for n in 128 256 512 1024; do
#  for j in 1 2 3 4 5; do
#    run 4 $n 3 1 $j 
#    run 4 $n 3 0 $j 
#  done
#done
#
## vary level
#for n in 1 2; do
#  for j in 1 2 3 4 5; do
#    run 4 1024 $n 1 $j 
#    run 4 1024 $n 0 $j 
#  done
#done
#
# vary thread
#for n in 16 64 256; do
#  for j in 1 2 3 4 5; do
#    run $n 1024 3 1 $j 
#    run $n 1024 3 0 $j 
#  done
#done
