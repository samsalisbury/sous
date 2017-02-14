# This is kind of a hack - in normal operation, Sous would block until its
# services had been accepted, but when bootstrapping, we need to wait for them
# to come up.
while [ $(cygnus -H http://192.168.99.100:7099/singularity | grep sous-server | wc -l) -lt 2 ]; do
  sleep 0.1
done
cygnus --env TASK_HOST --env PORT0 http://192.168.99.100:7099/singularity

leftport=$(cygnus --env PORT0 http://192.168.99.100:7099/singularity | grep 'sous-server.*left' | awk '{ print $3 }')
rightport=$(cygnus --env PORT0 http://192.168.99.100:7099/singularity | grep 'sous-server.*right' | awk '{ print $3 }')
serverURL=http://192.168.99.100:$leftport

until curl -s -I $serverURL; do
  sleep 0.1
done
sous config Server "$serverURL"
echo "Server URL is:" $(sous config Server)

ETAG=$(curl -s -v http://192.168.99.100:$leftport/servers 2>&1 | sed -n '/Etag:/{s/.*: //; P; }')
echo $ETAG
sed "s/LEFTPORT/$leftport/; s/RIGHTPORT/$rightport/" < ~/templated-configs/servers.json > ~/servers.json
cat ~/servers.json
curl -v -X PUT -H "If-Match: ${ETAG//[$'\t\r\n ']}" -H "Content-Type: application/json" "${serverURL}/servers" --data "$(< ~/servers.json)"
curl -s "${serverURL}/servers"
