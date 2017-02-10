	sous config
	cat ~/.config/sous/config.yaml
	git clone ssh://root@192.168.99.100:2222/repos/sous-server
	pushd sous-server
	export SOUS_USER_NAME=test SOUS_USER_EMAIL=test@test.com
	export SOUS_SERVER= SOUS_STATE_LOCATION=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing415154522/gdm
	sous init -v -d
	sous manifest get | sed '/left:/{
		a\
	\    env:
	  a\
	\      GDM_REPO: "ssh://root@192.168.99.100:2222/repos/gdm"
	  a\
	\      CLUSTER_NAME: left
  }
	/right:/{
		a\
	\    env:
	  a\
	\      GDM_REPO: "ssh://root@192.168.99.100:2222/repos/gdm"
	  a\
	\      CLUSTER_NAME: right
	}
	' > ~/sous-server.yaml
	cat ~/sous-server.yaml
	sous manifest set < ~/sous-server.yaml

	# Last minute config
	cat Dockerfile
	cp ~/dot-ssh/git_pubkey_rsa key_sous@example.com
	cp /Users/jlester/golang/src/github.com/opentable/sous/dev_support/$(readlink /Users/jlester/golang/src/github.com/opentable/sous/dev_support/sous_linux) .
	ls -la
	ssh-keyscan -p 2222 192.168.99.100 > known_hosts
	cat known_hosts

	git add key_sous@example.com known_hosts sous
	git commit -am "Adding ephemeral files"
	git tag -am "0.0.2" 0.0.2
	git push
	git push --tags
	sous context
	pwd
	sous build
	# We expect to see 'Sous is running ... in workstation mode' here:
	sous deploy -d -v -cluster left
	sous deploy -cluster right
	unset SOUS_SERVER
	unset SOUS_STATE_LOCATION
	popd
	