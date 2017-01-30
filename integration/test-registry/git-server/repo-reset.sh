#!/bin/bash

cd "$(dirname "$0")"
ssh -F ./ssh-config -i ./git_pubkey_rsa -p 2222 root@192.168.99.100 "/reset-repos"
