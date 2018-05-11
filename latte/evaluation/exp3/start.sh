source ./env.sh

ulimit -n 90000
num=${1:-1}

x=3
for n in `seq 1 $num`; do
  safeport=$((7776+n)) 
  listenport=$((19851+n))

  ssh -t compute4 docker run -dt --ulimit nofile=90000:90000 --rm -p ${safeport}:7777 -e RIAK_IP=10.10.1.$x --name safe$n safe
  ./metaserver --addr 10.10.1.3:8087 --addr 10.10.1.4:8087 --addr 10.10.1.5:8087 --safe compute4:$safeport --listen 0.0.0.0:$listenport  >perflog-meta-$n 2>&1 &
  # start safe container
  x=$((x+1))
  if [ $x -gt 5 ]; then
    x=3
  fi

done

