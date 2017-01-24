git clone ssh://root@192.168.99.100:2222/gdm
cat templated-configs/defs.yaml | tee gdm/defs.yaml
pushd gdm
git commit -am "Adding defs.yaml"
git push
popd
