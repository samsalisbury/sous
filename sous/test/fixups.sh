docker-machine ssh default sudo mkdir -p /etc/docker/certs.d/192.168.99.100:5000
docker-machine scp testing.crt default:/tmp
docker-machine ssh default sudo mv /tmp/testing.crt /etc/docker/certs.d/192.168.99.100:5000
