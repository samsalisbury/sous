#!/bin/bash

cd "$(dirname "$0")"
ssh -F ./ssh-config -i ./git_pubkey_rsa -p 2222 root@192.168.99.100 "cd /; tar zc repos > repos.tgz"
scp -F ./ssh-config -i ./git_pubkey_rsa -P 2222 root@192.168.99.100:/repos.tgz .
