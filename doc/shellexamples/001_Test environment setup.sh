# These steps are required by the Sous integration tests
# They're analogous to run-of-the-mill workstation maintenance.

env
mkdir -p /tmp/sous-cli-testing428272991/home/go/{src,bin}
go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
cd /home/judson/golang/src/github.com/opentable/sous
go install . #install the current sous project
cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/
cd /tmp/sous-cli-testing428272991
chmod go-rwx -R ~/dot-ssh
chmod +x -R ~/bin/*
ssh -o ConnectTimeout=1 -o PasswordAuthentication=no -F "${HOME}/dot-ssh/config" root@127.0.0.1 -p 2222 /reset-repos < /dev/null
