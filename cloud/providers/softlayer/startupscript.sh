#!/bin/bash

# ref: https://stackoverflow.com/a/2831449/244009
cd /tmp
/usr/bin/wget -O kubeadm.sh https://api.service.softlayer.com/rest/v3/SoftLayer_Resource_Metadata/UserMetadata.txt
chmod +x kubeadm.sh
nohup ./kubeadm.sh > /dev/null 2>&1 &

exit 0
