#!/usr/bin/env sh

set -x

GDM_DIR1=/home/vagrant/gdm-cluster1-local
GDM_DIR2=/home/vagrant/gdm-cluster2-local
REMOTE_DIR=/home/vagrant/gdm-remote

CONFIG_DIR_SERVER1=/home/vagrant/.config/sous-server1
CONFIG_DIR_SERVER2=/home/vagrant/.config/sous-server2

git config --global user.name="Dev Sous Server"
git config --global user.email=sous-server-dev@example.com

# New test GDM
sous new-test-gdm -checkout-dir $GDM_DIR1

# Init the new test GDM
( cd $GDM_DIR1 && git init && git add -A && git commit -m 'new-test-gdm' && \
	git remote add origin $REMOTE_DIR )

# Copy to remote dir.
cp -r $GDM_DIR1 $REMOTE_DIR

(
	echo "Starting Sous Server 1"
	export SOUS_CONFIG_DIR=$CONFIG_DIR_SERVER1
	sous config StateLocation $GDM_DIR1
	sous server -cluster cluster1 -gdm-repo $REMOTE_DIR -listen :4646 > /var/log/sous-server1.log 2>&1 &
	echo "Done starting Sous Server 1"
)

(
	echo "Starting Sous Server 2"
	export SOUS_CONFIG_DIR=$CONFIG_DIR_SERVER2
	sous config StateLocation $GDM_DIR2
	sous server -cluster cluster2 -gdm-repo $REMOTE_DIR -listen 5757 > /var/log/sous-server2.log 2>&1 &
	echo "Done starting Sous Server 2"
)



