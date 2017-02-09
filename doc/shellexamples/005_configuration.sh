	cygnus --env TASK_HOST --env PORT0 http://192.168.99.100:7099/singularity
	serverURL=$(cygnus --env TASK_HOST --env PORT0 http://192.168.99.100:7099/singularity | grep 'sous-server.*left' | awk '{ print "http://" $3 ":" $4 }')
	sous config Server "$serverURL"
	echo -n "Server URL is: "
	sous config Server
	