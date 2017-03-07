cd
rm -rf sous-demo
git clone ssh://root@192.168.99.100:2222/repos/sous-demo
cd sous-demo
git tag -am 'Release!' 0.0.24
git push --tags

# We will make this deploy fail by asking for too many resources.
sous manifest get > demo_manifest.yaml
cat demo_manifest.yaml
# Set CPUs to redonkulous.
sed 's/^      cpus.*$/      cpus: "30"/g' demo_manifest.yaml > demo_manifest_toobig.yaml
cat demo_manifest_toobig.yaml
sous manifest set < demo_manifest_toobig.yaml
sous build
