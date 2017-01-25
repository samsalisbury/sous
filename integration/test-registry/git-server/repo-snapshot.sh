#!/bin/bash

ssh -F ./ssh-config -i ./git_pubkey_rsa root@192.168.99.100 -p 2222 "cd /; tar zvc repos" > repos.tgz
