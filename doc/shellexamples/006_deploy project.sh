	git clone ssh://root@192.168.99.100:2222/repos/sous-demo
	cd sous-demo
	git tag -a 0.0.23
	git push --tags
	sous init
	sous build
	sous deploy -cluster left
	