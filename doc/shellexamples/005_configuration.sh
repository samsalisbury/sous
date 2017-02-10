	# This is kind of a hack - in normal operation, Sous would block until its
	# services had been accepted, but when bootstrapping, we need to wait for them
	# to come up.
	while [ $(cygnus -H http://192.168.99.100:7099/singularity | grep sous-server | wc -l) -lt 2 ]; do
	  sleep 0.1
	done
	cygnus --env TASK_HOST --env PORT0 http://192.168.99.100:7099/singularity
	serverURL=http://192.168.99.100:$(cygnus --env PORT0 http://192.168.99.100:7099/singularity | grep 'sous-server.*left' | awk '{ print $3 }')
	sous config Server "$serverURL"
	echo "Server URL is:" $(sous config Server)
	