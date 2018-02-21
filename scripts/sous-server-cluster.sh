#!/usr/bin/env bash

# Create an isolated cluster of sous server instances, each with their own
# configuration & local state checkout.
#
# In light of the cluster definition, write the correct SiblingURLs to each
# server's configuration, and make a single commit to the cloned GDM to add
# each cluster defined.

CLUSTERS='cluster1 cluster2 cluster3'
SING_URL='http://192.168.99.100:7099/singularity'
DOCKER_REG='http://192.168.99.100:5000'
BASE_DIR="$HOME/.sous/test-clusters"

rm -rf "$BASE_DIR"

echo "Clusters to be created:"
for CL in $CLUSTERS; do
	echo "name: $CL; singularity: $SING_URL"
done

echo

GDM_REMOTE_DIR="$BASE_DIR/remote_gdm"
mkdir -p "$GDM_REMOTE_DIR"
rm -rf "$GDM_REMOTE_DIR"
echo "Writing test GDM to $GDM_REMOTE_DIR"

# COMMA_CLUSTERS is needed for sous new-test-gdm call
COMMA_CLUSTERS="${CLUSTERS// /,}"
sous new-test-gdm -dir "$GDM_REMOTE_DIR" -clusters "$COMMA_CLUSTERS" -singularity "$SING_URL" -docker-reg "$DOCKER_REG"

cd "$GDM_REMOTE_DIR" || { echo "Failed to CD to $GDM_REMOTE_DIR"; exit 1; }

git init && git add defs.yaml && git commit -m "initial commit"

for CL in $CLUSTERS; do
	echo "name: $CL; singularity: $SING_URL"
	INST_DIR="$BASE_DIR/$CL"
	CONFIG_DIR="$INST_DIR/config"
	CONFIG_FILE="$CONFIG_DIR/config.yaml"
	STATE_DIR="$INST_DIR/state"
	git clone "$GDM_REMOTE_DIR" "$STATE_DIR"
	mkdir -p "$CONFIG_DIR" "$STATE_DIR"
	echo 'StateLocation: '"$STATE_DIR"'
Database:
  DBName: ""
  User: ""
  Password: ""
  Host: ""
  Port: ""
  SSL: false
SiblingURLs: {}
BuildStateDir: ""
Docker:
  RegistryHost: '$DOCKER_REG'
  DatabaseDriver: sqlite3_sous
  DatabaseConnection: file:dummy.db?mode=memory&cache=shared
Logging:
  Basic:
    Level: "critical"
User:
  Name: Vagrant Sous Server
  Email: vagrant-sous-server@example.com
MaxHTTPConcurrencySingularity: 10' > "$CONFIG_FILE"
done

echo "Starting cluster..."

i=0
PIDS=""
for CL in $CLUSTERS; do
	SOUS_CONFIG_DIR="$INST_DIR/config" sous server -cluster "$CL" -listen ":555$i" > "/var/log/sous-server-$CL" 2>&1 &
	PIDS="$PIDS$! "
	i=$(( i + 1 ))
done

kill_servers() {
	kill -15 $PIDS
}

trap kill_servers SIGINT

echo "Sous servers running; press ctrl+c to kill."

wait

