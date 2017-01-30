git clone ssh://root@192.168.99.100:2222/repos/gdm
cat templated-configs/defs.yaml | tee gdm/defs.yaml
pushd gdm
git add defs.yaml
git commit -am "Adding defs.yaml"
git push
popd
