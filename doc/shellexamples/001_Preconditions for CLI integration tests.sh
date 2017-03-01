date
if [ -n "$GOROOT" ]; then
	mkdir -p $GOROOT
fi
go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
if echo "http://192.168.99.100:7099/singularity" | egrep -q '192.168|127.0.0'; then
	/Users/ssalisbury/go/src/github.com/opentable/sous/integration/test-registry/clean-singularity.sh http://192.168.99.100:7099/singularity
fi
cygnus -H http://192.168.99.100:7099/singularity
ls /Users/ssalisbury/go/src/github.com/opentable/sous/dev_support
