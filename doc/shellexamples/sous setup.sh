git clone ssh://root@192.168.99.100:2222/repos/sous-server
pushd sous-server
sous init
sous build
# We expect to see 'Sous is running ... in workstation mode' here:
SOUS_SERVER= SOUS_STATE_LOCATION=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing394335736/gdm sous deploy -cluster left
SOUS_SERVER= SOUS_STATE_LOCATION=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing394335736/gdm sous deploy -cluster right
popd
