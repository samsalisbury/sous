	go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
	if echo "http://192.168.99.100:7099/singularity" | egrep -q '192.168|127.0.0'; then
		/Users/jlester/golang/src/github.com/opentable/sous/integration/test-registry/clean-singularity.sh http://192.168.99.100:7099/singularity
	fi
	cygnus -H http://192.168.99.100:7099/singularity
	ls /Users/jlester/golang/src/github.com/opentable/sous/dev_support
	