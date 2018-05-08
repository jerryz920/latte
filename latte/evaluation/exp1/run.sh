
	#flag.Var(&Addresses, "addr", "metadata service")
	#flag.IntVar(&Nthread, "nthread", 1, "num of cocurrent thread")
	#flag.IntVar(&NumVM, "nvm", 200, "num of vm instance")
	#flag.IntVar(&NumLevel, "nlevel", 3, "num of level")
	#flag.BoolVar(&RandAccess, "rand", false, "random access")
	#flag.IntVar(&Round, "round", 1, "random access")

msdir="/openstack/go/src/github.com/jerryz920/conferences/latte/proxy"
riakdir="/openstack/safe/uber-safe/cluster-scripts"
#Nthread="4 16 64 256"
Nthread=4


config() {
  args=""
  for n in 1 2 3 4; do
    args="$args --addr compute4:$((19850+n))"
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
  echo "starting $* exps"
  echo cd $riakdir
  echo bash allrestart.sh
  sleep 10
  echo ssh compute4 "cd $msdir; bash stop.sh 4; bash start.sh 4"
  args=`config $*`
  sleep 10
  echo ./exp1 $args
}

for n in 4; do
  for j in 1; do
    run 4 $n 3 1 $j >> perf-log-test
    run 4 $n 3 0 $j >> perf-log-test
  done
done

# vary thread
for n in 2; do
  for j in 1; do
    run $n 4 3 1 $j >> perf-log-test
    run $n 4 3 0 $j >> perf-log-test
  done
done

# vary level
for n in 1; do
  for j in 1; do
    run 4 4 $n 1 $j >> perf-log-test
    run 4 4 $n 0 $j >> perf-log-test
  done
done

# vary nvm
#for n in 128 256 512 1024; do
#  for j in 1 2 3 4 5; do
#    run 4 $n 3 1 $j >> perf-log
#    run 4 $n 3 0 $j >> perf-log
#  done
#done
#
## vary thread
#for n in 16 64 256; do
#  for j in 1 2 3 4 5; do
#    run $n 1024 3 1 $j >> perf-log
#    run $n 1024 3 0 $j >> perf-log
#  done
#done
#
## vary level
#for n in 1 2; do
#  for j in 1 2 3 4 5; do
#    run 4 1024 $n 1 $j >> perf-log
#    run 4 1024 $n 0 $j >> perf-log
#  done
#done
