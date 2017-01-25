# These steps are required by the Sous integration tests
# They're analogous to run-of-the-mill workstation maintenance.

env
mkdir -p /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing394335736/home/go/{src,bin}
go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
cd /Users/jlester/golang/src/github.com/opentable/sous
go install . #install the current sous project
cp -a integration/test-homedir/* "$HOME"
cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/
cd /var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing394335736
cp templated-configs/ssh-config ~/dot-ssh/config
chmod go-rwx -R ~/dot-ssh
git config --global --add user.name "Integration Tester"
git config --global --add user.email "itester@example.com"
hash -r
