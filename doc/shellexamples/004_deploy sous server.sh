sous config
cat ~/.config/sous/config.yaml
git clone ssh://root@192.168.99.100:2222/repos/sous-server
pushd sous-server
export SOUS_USER_NAME=test SOUS_USER_EMAIL=test@test.com
export SOUS_SERVER= SOUS_STATE_LOCATION=/tmp/sous-work/gdm

sous init
sous manifest get
sous manifest set < ~/templated-configs/sous-server.yaml
sous manifest get # demonstrating this got to GDM

# Last minute config
cat Dockerfile
cp ~/dot-ssh/git_pubkey_rsa key_sous@example.com
cp /Users/ssalisbury/go/src/github.com/opentable/sous/dev_support/$(readlink /Users/ssalisbury/go/src/github.com/opentable/sous/dev_support/sous_linux) .
cp /Users/ssalisbury/go/src/github.com/opentable/sous/integration/test-registry/docker-registry/testing.crt docker.crt

ls -a
ssh-keyscan -p 2222 192.168.99.100 > known_hosts

git add key_sous@example.com known_hosts sous
git commit -am "Adding ephemeral files"
git tag -am "0.0.2" 0.0.2
git push
git push --tags

sous build
sous deploy -cluster left # We expect to see 'Sous is running ... in workstation mode' here:
sous deploy -cluster right
unset SOUS_SERVER
unset SOUS_STATE_LOCATION
popd
