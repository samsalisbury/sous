sous config
cat ~/.config/sous/config.yaml
git clone ssh://root@127.0.0.1:2222/repos/sous-server
pushd sous-server
SOUS_SERVER= SOUS_STATE_LOCATION=/tmp/sous-cli-testing428272991/gdm sous init -v -d

# Last minute config
cat Dockerfile
cp ~/dot-ssh/git_pubkey_rsa key_sous@example.com
cp $(which sous) .
ls -la
ssh-keyscan -p 2222 127.0.0.1 > known_hosts
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
SOUS_SERVER= SOUS_STATE_LOCATION=/tmp/sous-cli-testing428272991/gdm sous deploy -cluster left
SOUS_SERVER= SOUS_STATE_LOCATION=/tmp/sous-cli-testing428272991/gdm sous deploy -cluster right
popd
