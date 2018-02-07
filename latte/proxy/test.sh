
# the error is about 62ms for timestamp
# control the test

# base = 2002
# for i = 1 - 20;
# post 1.1.1.1 base - 2 + 3 * i, base - 1 + 3 * i, base + 3 * i with same git source
# for kill = 2000, 2001, 2002; do
#   boot mysql with 10.10.1.36 test port = kill, ip = 1.1.1.1, expose 1999->19999, short.sh
#   boot mysql-runner with /run-short.sh
#   timestamp
#   poison 2000
#
IAAS_IP="152.3.145.38:444"

base=${1:-53018}
base2=$((base+100))
msg=`curl http://10.10.1.39:19851/postInstanceSet -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"test1\", \"instance-image-hash\", \"image\", \"1.1.1.1:$base-$base2\", \"noconfig\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:19851/updateSubjectSet -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"$inst_id\"] }"
sleep 0.1


curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash\", \"noconfig\"]}"
sleep 0.1
curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash\", \"noconfig\", \"git://github.com/jerryz920/docker\"]}"
sleep 0.1
curl http://10.10.1.39:19851/postObjectAcl -d "{ \"principal\": \"alice\",  \"otherValues\": [\"alice:object1\", \"git://github.com/jerryz920/mysql\"] }"
sleep 0.1
curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$base\"]}"
sleep 0.1
curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$base-$base2\"]}"
sleep 0.1
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$base\", \"git://github.com/jerryz920/docker\"]}"
sleep 0.1
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$((base+5))\", \"git://github.com/jerryz920/docker\"]}"
sleep 0.1
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$base2\", \"git://github.com/jerryz920/docker\"]}"
sleep 0.1
echo
echo should be rejected
sleep 0.1
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$base\", \"alice:object1\"]}"
sleep 0.1
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$((base+5))\", \"alice:object1\"]}"
sleep 0.1
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$base2\", \"alice:object1\"]}"


cbase=$((base+10))
cbase2=$((base+50))

msg=`curl http://10.10.1.39:19851/postInstanceSet -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"test2\", \"instance-image-hash-2\", \"image\", \"1.1.1.1:$cbase-$cbase2\", \"noconfig\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:19851/updateSubjectSet -d "{ \"principal\": \"1.1.1.1:$cbase-$cbase2\",  \"otherValues\": [\"$inst_id\"] }"
#curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"instance-image-hash2\"]}"
curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash2\", \"noconfig\"]}"
#curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"instance-image-hash2\", \"git://github.com/jerryz920/notmysql\"]}"
curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash2\", \"noconfig\", \"git://github.com/jerryz920/notmysql\"]}"
curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$cbase\"]}"
curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$cbase2\"]}"


curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$cbase\", \"git://github.com/jerryz920/notmysql\"]}"
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$((cbase+5))\", \"git://github.com/jerryz920/notmysql\"]}"
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$cbase2\", \"git://github.com/jerryz920/notmysql\"]}"

echo
echo should be rejected
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$cbase\", \"alice:object1\"]}"
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$((cbase+5))\", \"alice:object1\"]}"
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$cbase2\", \"alice:object1\"]}"


c2base=$((cbase))
c2base2=$((cbase+6))

msg=`curl http://10.10.1.39:19851/postInstanceSet -d "{ \"principal\": \"1.1.1.1:$cbase-$cbase2\",  \"otherValues\": [\"test3\", \"instance-image-hash-3\", \"image\", \"1.1.1.1:$c2base-$c2base2\", \"noconfig\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:19851/updateSubjectSet -d "{ \"principal\": \"1.1.1.1:$c2base-$c2base2\",  \"otherValues\": [\"$inst_id\"] }"
#curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"1.1.1.1:$cbase-$cbase2\",  \"otherValues\": [\"instance-image-hash3\"]}"
curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash3\", \"noconfig\"]}"
#curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"1.1.1.1:$cbase-$cbase2\",  \"otherValues\": [\"instance-image-hash3\", \"git://github.com/jerryz920/mysql\"]}"
curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash3\", \"noconfig\", \"git://github.com/jerryz920/mysql\"]}"
curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$c2base\"]}"
curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$c2base2\"]}"


curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$c2base\", \"git://github.com/jerryz920/mysql\"]}"
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$((c2base+5))\", \"git://github.com/jerryz920/mysql\"]}"
curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$c2base2\", \"git://github.com/jerryz920/mysql\"]}"

echo
echo should be accepted
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$c2base\", \"alice:object1\"]}"
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$((c2base+5))\", \"alice:object1\"]}"
curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$c2base2\", \"alice:object1\"]}"

#curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$c2base\", \"alice:object1\"]}"
#curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$((c2base+5))\", \"alice:object1\"]}"
#curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\",\"otherValues\": [\"1.1.1.1:$c2base2\", \"alice:object1\"]}"

#msg=`curl http://10.10.1.39:19851/retractInstanceSet -d "{ \"principal\": \"1.1.1.1:$cbase-$cbase2\",  \"otherValues\": [\"test3\", \"instance-image-hash-3\", \"image\", \"1.1.1.1:$c2base-$c2base2\", \"noconfig\"] }"`
#echo $msg | tee key
#inst_id=`python id.py <key`
#curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$c2base\"]}"
#
#
#msg=`curl http://10.10.1.39:19851/retractInstanceSet -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"test2\", \"instance-image-hash-2\", \"image\", \"1.1.1.1:$cbase-$cbase2\", \"noconfig\"] }"`
#echo $msg | tee key
#inst_id=`python id.py <key`
#curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$cbase\"]}"
#
#msg=`curl http://10.10.1.39:19851/retractInstanceSet -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"test1\", \"instance-image-hash\", \"image\", \"1.1.1.1:$base-$base2\", \"noconfig\"] }"`
#echo $msg | tee key
#inst_id=`python id.py <key`
#curl http://10.10.1.39:19851/attestInstance -d "{ \"principal\": \"152.3.145.138:4144\",  \"bearerRef\": \"$inst_id\", \"otherValues\": [\"1.1.1.1:$base\"]}"


#
#
##  msg=`curl http://10.10.1.39:19851/postInstanceSet -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"test1\", \"instance-image-hash\", \"image\", \"1.1.1.1:100-200\", \"noconfig\"] }"`
##  echo $msg | tee key
##  inst_id=`python id.py <key`
##  curl http://10.10.1.39:19851/updateSubjectSet -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"$inst_id\"] }"
##  curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash\"]}"
##  curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash\", \"git://github.com/jerryz920/mysql\"]}"
##
##
##  curl http://10.10.1.39:19851/postObjectAcl -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"object1\", \"git://github.com/jerryz920/mysql\"] }"
##  curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$base-$base2\", \"git://github.com/jerryz920/mysql\"]}"
##  curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$base-$base2\", \"object1\"]}"
##
##  msg=`curl http://10.10.1.39:19851/postInstanceSet -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"test2\", \"instance-image-hash-2\", \"image\", \"1.1.1.1:$base-$base2\", \"noconfig\"] }"`
##  echo $msg | tee key
##  inst_id=`python id.py <key`
##  curl http://10.10.1.39:19851/updateSubjectSet -d "{ \"principal\": \"1.1.1.1:$base-$base2\",  \"otherValues\": [\"$inst_id\"] }"
##  curl http://10.10.1.39:19851/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash2\"]}"
##  curl http://10.10.1.39:19851/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"instance-image-hash\", \"git://github.com/jerryz920/notmysql\"]}"
##
##
##  curl http://10.10.1.39:19851/attestAppProperty -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$base-$base2\", \"git://github.com/jerryz920/mysql\"]}"
##  curl http://10.10.1.39:19851/appAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\",  \"otherValues\": [\"1.1.1.1:$base-$base2\", \"object1\"]}"
#
#
#
## base = 20002
## for i = 1 - 20;
## post 1.1.1.1 base - 2 + 3 * i, base - 1 + 3 * i, base + 3 * i with same git source
## for kill = 2000, 2001, 2002; do
##   boot mysql with 10.10.1.36 test port = kill, ip = 1.1.1.1, expose 1999->19999, long.sh
##   boot mysql-runner with /run-long.sh
##   timestamp
##   poison 2000
##
#
