# These steps are required by the Sous integration tests
# They're analogous to run-of-the-mill workstation maintenance.

env
export SOUS_EXTRA_DOCKER_CA=/Users/jlester/golang/src/github.com/opentable/sous/integration/test-registry/docker-registry/testing.crt
mkdir -p /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing266901424/home/go/{src,bin}
GOPATH=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing266901424/home/go go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
cd /Users/jlester/golang/src/github.com/opentable/sous
go install . #install the current sous project
cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/
cd /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing266901424
chmod go-rwx -R ~/dot-ssh
chmod +x -R ~/bin/*
ssh -o ConnectTimeout=1 -o PasswordAuthentication=no -F "${HOME}/dot-ssh/config" root@192.168.99.100 -p 2222 /reset-repos < /dev/null
