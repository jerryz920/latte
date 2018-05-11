source ./env.sh

num=${1:-1}
for n in `seq 1 $num`; do
  ssh -t compute4 docker rm -f safe$n
done
  killall metaserver
