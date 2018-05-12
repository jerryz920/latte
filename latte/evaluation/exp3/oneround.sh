for n in final; do
  echo starting round $n
source utils.sh
bash buildsfrom.sh builds-log-$n
bash connection.sh connection-log-$n
bash container-isolation.sh isolation-log-$n
bash launches.sh launch-log-$n
bash launch-guard.sh guard-log-$n
bash membership.sh member-log-$n
bash quality.sh quality-log-$n
#bash exp3.sh  caching-log-$n
echo one round $n finish!
done
