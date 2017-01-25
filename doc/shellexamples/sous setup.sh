git clone ssh://root@192.168.99.100:2222/repos/sous-server
pushd sous-server
sous build
SOUS_SERVER= SOUS_STATE_LOCATION=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing399335466/gdm sous deploy -cluster left
SOUS_SERVER= SOUS_STATE_LOCATION=/var/folders/sp/wllf_wh92p725fl4vz92mrn16vkfds/T/sous-cli-testing399335466/gdm sous deploy -cluster right
popd
