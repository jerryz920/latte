#python3 gen-data.py -r `seq 70 79` -m 1 -o all -t 32&
#python3 gen-data.py -r `seq 70 79` -m 0 -o all -t 32&
#python3 gen-data.py -r `seq 70 79` -m 1 -o checkFetch -t 32&
#python3 gen-data.py -r `seq 70 79` -m 0 -o checkFetch -t 32&
#python3 gen-data.py -r `seq 80 89` -m 1 -o all -t 128&
#python3 gen-data.py -r `seq 80 89` -m 0 -o all -t 128&
#python3 gen-data.py -r `seq 80 89` -m 1 -o checkFetch -t 128&
#python3 gen-data.py -r `seq 80 89` -m 0 -o checkFetch -t 128&
#python3 gen-data.py -r `seq 100 109` -m 1 -o all -t 256&
#python3 gen-data.py -r `seq 100 109` -m 0 -o all -t 256&
#python3 gen-data.py -r `seq 100 109` -m 1 -o checkFetch -t 256&
#python3 gen-data.py -r `seq 100 109` -m 0 -o checkFetch -t 256&


#python3 gen-data.py -n 512 -r `seq 200 209` -m 1 -o all -t 128&
#python3 gen-data.py -n 512 -r `seq 200 209` -m 0 -o all -t 128&
#python3 gen-data.py -n 512 -r `seq 200 209` -m 1 -o checkFetch -t 128&
#python3 gen-data.py -n 512 -r `seq 200 209` -m 0 -o checkFetch -t 128&
#
#python3 gen-data.py -n 1024 -r `seq 210 219` -m 1 -o all -t 128&
#python3 gen-data.py -n 1024 -r `seq 210 219` -m 0 -o all -t 128&
#python3 gen-data.py -n 1024 -r `seq 210 219` -m 1 -o checkFetch -t 128&
#python3 gen-data.py -n 1024 -r `seq 210 219` -m 0 -o checkFetch -t 128&
#
#python3 gen-data.py -n 2048 -r `seq 220 229` -m 1 -o all -t 128&
#python3 gen-data.py -n 2048 -r `seq 220 229` -m 0 -o all -t 128&
#python3 gen-data.py -n 2048 -r `seq 220 229` -m 1 -o checkFetch -t 128&
#python3 gen-data.py -n 2048 -r `seq 220 229` -m 0 -o checkFetch -t 128&

python3 gen-data.py -l 1 -r `seq 230 239` -m 1 -o all -t 128&
python3 gen-data.py -l 1 -r `seq 230 239` -m 0 -o all -t 128&
python3 gen-data.py -l 1 -r `seq 230 239` -m 1 -o checkFetch -t 128&
python3 gen-data.py -l 1 -r `seq 230 239` -m 0 -o checkFetch -t 128&

python3 gen-data.py -l 2 -r `seq 240 249` -m 1 -o all -t 128&
python3 gen-data.py -l 2 -r `seq 240 249` -m 0 -o all -t 128&
python3 gen-data.py -l 2 -r `seq 240 249` -m 1 -o checkFetch -t 128&
python3 gen-data.py -l 2 -r `seq 240 249` -m 0 -o checkFetch -t 128&

for n in `jobs -p`; do
  wait $n
done
