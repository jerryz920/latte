
num=${1:-1}

for n in `seq 1 $num`; do
docker rm -f safe$n
killall proxy
done
