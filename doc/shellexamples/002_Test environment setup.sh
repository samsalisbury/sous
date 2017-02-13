	# These steps are required by the Sous integration tests
	# They're analogous to run-of-the-mill workstation maintenance.

	cd /Users/jlester/golang/src/github.com/opentable/sous
	env
	export SOUS_EXTRA_DOCKER_CA=/Users/jlester/golang/src/github.com/opentable/sous/integration/test-registry/docker-registry/testing.crt
	mkdir -p /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing785712611/home/go/{src,bin}

	### This build gives me trouble in tests...
	### xgo does something weird and different with it's dep-cache dir
	# GOPATH=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing785712611/home/go make linux_build # we need Sous built for linux for the server
	go install . #install the current sous project
	cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/

	cd /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing785712611
	chmod go-rwx -R ~/dot-ssh
	chmod +x -R ~/bin/*
	ssh -o ConnectTimeout=1 -o PasswordAuthentication=no -F "${HOME}/dot-ssh/config" root@192.168.99.100 -p 2222 /reset-repos < /dev/null
	