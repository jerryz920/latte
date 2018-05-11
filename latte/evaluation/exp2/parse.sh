


python3 gen-data.py --records `seq 16 20`  --op "postCluster" --member 30 --check any&
python3 gen-data.py --records `seq 16 20`  --op "postMembership" --member 30 --check any&
python3 gen-data.py --records `seq 16 20`  --op "postAckMembership" --member 30 --check any&
python3 gen-data.py --records `seq 16 20`  --op "checkFetch" --member 30 --check vmcheck-master&
python3 gen-data.py --records `seq 16 20`  --op "checkFetch" --member 30 --check ctncheck-master&
python3 gen-data.py --records `seq 16 20`  --op "checkFetch" --member 30 --check proccheck-master&
python3 gen-data.py --records `seq 16 20`  --op "checkFetch" --member 30 --check vmcheck&
python3 gen-data.py --records `seq 16 20`  --op "checkFetch" --member 30 --check ctncheck&
python3 gen-data.py --records `seq 16 20`  --op "checkFetch" --member 30 --check proccheck&

for n in `jobs -p`; do
  wait $n
done
