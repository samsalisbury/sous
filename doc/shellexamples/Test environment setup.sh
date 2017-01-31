# These steps are required by the Sous integration tests
# They're analogous to run-of-the-mill workstation maintenance.

env
# set -P
mkdir -p /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing375798555/home/go/{src,bin}
go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
cd /Users/jlester/golang/src/github.com/opentable/sous
go install . #install the current sous project
cp -a integration/test-homedir/* "$HOME"
cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/
cd /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing375798555
cp templated-configs/ssh-config ~/dot-ssh/config
chmod go-rwx -R ~/dot-ssh
git config --global --add user.name "Integration Tester"
git config --global --add user.email "itester@example.com"
ssh -o ConnectTimeout=1 -o PasswordAuthentication=no -F "${HOME}/dot-ssh/config" root@192.168.99.100 -p 2222 /reset-repos < /dev/null
echo SOME STUFF HERE GOT ET BY SSH
