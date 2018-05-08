
num=${1:-1}

for n in `seq 1 $num`; do
  safeport=$((7776+n)) 
  listenport=$((19850+n))
  docker run -dt --rm -p $safeport:7777 -e RIAK_IP=10.10.1.4 --name safe$n safe
  ./proxy --addr 10.10.1.3:8087 --addr 10.10.1.4:8087 --addr 10.10.1.5:8087 --safe localhost:$safeport --listen 0.0.0.0:$listenport --debug 2>&1 | tee perflog-$n
  # start safe container
done

