	cygnus --env TASK_HOST --env PORT0 http://192.168.99.100:7099/singularity
	serverURL=http://192.168.99.100:$(cygnus--env PORT0 http://192.168.99.100:7099/singularity | grep 'sous-server.*left' | awk '{ print $3 }')
	sous config Server "$serverURL"
	echo "Server URL is:" $(sous config Server)
	