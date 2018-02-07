IAAS_IP="152.3.145.38:444"
ATTEST_IP="152.3.145.138:4144"
echo testing membership
base=${1:-57018}
base2=$((base+100))
cbase=$((base+10))
cbase2=$((base+50))
c2base=$((cbase))
c2base2=$((cbase+6))

curl http://10.10.1.39:7777/postObjectAcl -d "{ \"principal\": \"bob\",  \"otherValues\": [\"bob:object1\", \"git://github.com/jerryz920/pio\"] }"
#curl http://10.10.1.39:7777/postObjectAcl -d "{ \"principal\": \"bob\",  \"otherValues\": [\"bob:object1\", \"git://github.com/jerryz920/docker\"] }"
curl http://10.10.1.39:7777/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"membership-image-hash\", \"*\", \"git://github.com/jerryz920/docker\"]}"
curl http://10.10.1.39:7777/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"membership-image-hash\", \"*\"]}"
curl http://10.10.1.39:7777/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"membership-image-hash2\", \"*\", \"git://github.com/jerryz920/spark\"]}"
curl http://10.10.1.39:7777/postAttesterImage -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"membership-image-hash2\", \"*\"]}"
curl http://10.10.1.39:7777/postImageProperty -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"membership-image-hash3\", \"*\", \"git://github.com/jerryz920/pio\"]}"

curl http://10.10.1.39:7777/postInstanceSet -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"2\", \"membership-image-hash\", \"image\", \"9.9.9.9:$base-$base2\", \"*\"] }" | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"9.9.9.9:$base-$base2\",  \"otherValues\": [\"$inst_id\"] }"

msg=`curl http://10.10.1.39:7777/postInstanceSet -d "{ \"principal\": \"9.9.9.9:$base-$base2\",  \"otherValues\": [\"2\", \"membership-image-hash2\", \"image\", \"9.9.9.9:$cbase-$cbase2\", \"*\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"9.9.9.9:$cbase-$cbase2\",  \"otherValues\": [\"$inst_id\"] }"
#curl http://10.10.1.39:7777/postAttesterImage -d "{ \"principal\": \"9.9.9.9:$base-$base2\",  \"otherValues\": [\"membership-image-hash2\"]}"

msg=`curl http://10.10.1.39:7777/postInstanceSet -d "{ \"principal\": \"9.9.9.9:$base-$base2\",  \"otherValues\": [\"3\", \"membership-image-hash3\", \"image\", \"9.9.9.9:$c2base-$c2base2\", \"*\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"9.9.9.9:$c2base-$c2base2\",  \"otherValues\": [\"$inst_id\"] }"



msg=`curl http://10.10.1.39:7777/postInstanceSet -d "{ \"principal\": \"$IAAS_IP\",  \"otherValues\": [\"3\", \"membership-image-hash\", \"image\", \"11.11.11.11:$base-$base2\", \"*\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"11.11.11.11:$base-$base2\",  \"otherValues\": [\"$inst_id\"] }"


msg=`curl http://10.10.1.39:7777/postInstanceSet -d "{ \"principal\": \"11.11.11.11:$base-$base2\",  \"otherValues\": [\"3\", \"membership-image-hash2\", \"image\", \"11.11.11.11:$cbase-$cbase2\", \"*\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"11.11.11.11:$cbase-$cbase2\",  \"otherValues\": [\"$inst_id\"] }"


msg=`curl http://10.10.1.39:7777/postInstanceSet -d "{ \"principal\": \"11.11.11.11:$base-$base2\",  \"otherValues\": [\"3\", \"membership-image-hash3\", \"image\", \"11.11.11.11:$c2base-$c2base2\", \"*\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"11.11.11.11:$c2base-$c2base2\",  \"otherValues\": [\"$inst_id\"] }"




msg=`curl http://10.10.1.39:7777/postWorkerSet -d "{ \"principal\": \"11.11.11.11:$c2base-$c2base2\",  \"otherValues\": [\"cluster1\", \"9.9.9.9:$((c2base+1))\", \"*\"] }"`
echo $msg | tee key
inst_id=`python id.py <key`
msg=`curl http://10.10.1.39:7777/updateSubjectSet -d "{ \"principal\": \"9.9.9.9:$c2base-$c2base2\",  \"otherValues\": [\"$inst_id\"] }"`
echo $msg | tee key
inst_id_worker=`python id.py <key`



curl http://10.10.1.39:7777/workerAccessesObject -d "{ \"principal\": \"152.3.145.138:4144\", \"bearerRef\":\"$inst_id_worker\", \"otherValues\": [\"9.9.9.9:$((c2base+1))\", \"bob:object1\"]}"
