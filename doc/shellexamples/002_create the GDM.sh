git clone ssh://root@192.168.99.100:2222/repos/gdm
cp ~/templated-configs/defs.yaml gdm/defs.yaml
cat gdm/defs.yaml
pushd gdm
cat ~/.config/git/config >> .git/config # Eh?
git config -l
git add defs.yaml
git commit -am "Adding defs.yaml"
git push
popd
