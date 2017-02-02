git clone ssh://root@127.0.0.1:2222/repos/gdm
cp ~/templated-configs/defs.yaml gdm/defs.yaml
cat gdm/defs.yaml
pushd gdm
cat ~/.config/git/config >> .git/config # Eh?
git add defs.yaml
git commit -am "Adding defs.yaml"
git push
popd
