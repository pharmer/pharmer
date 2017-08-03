#!/usr/bin/env python


# http://stackoverflow.com/a/14050282
def check_antipackage():
    from sys import version_info
    sys_version = version_info[:2]
    found = True
    if sys_version < (3, 0):
        # 'python 2'
        from pkgutil import find_loader
        found = find_loader('antipackage') is not None
    elif sys_version <= (3, 3):
        # 'python <= 3.3'
        from importlib import find_loader
        found = find_loader('antipackage') is not None
    else:
        # 'python >= 3.4'
        from importlib import util
        found = util.find_spec('antipackage') is not None
    if not found:
        print('Install missing package "antipackage"')
        print('Example: pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage')
        from sys import exit
        exit(1)
check_antipackage()


# ref: https://github.com/ellisonbg/antipackage
import antipackage
from github.appscode.libbuild import libbuild

import datetime
import json
import os
import os.path
import subprocess
import sys
import time
from os.path import expandvars
import make # make.py

# appctl cluster create --provider=aws --nodes=t2.micro:2 --zone=us-west-2a os56 < $GOPATH/src/appscode.com/ark/conf/aws.csv
# appctl cluster create --provider=gce --nodes=n1-standard-1:2 kf1
# appctl cluster delete os56
# adminctl ssh -k=mailtest -c=os56 -i=ec2-54-86-46-157.compute-1.amazonaws.com

REPO_ROOT = expandvars('$GOPATH') + '/src/appscode.com/ark'
KLUSTER_PATH = REPO_ROOT + '/kluster.json'
BUILD_METADATA = libbuild.metadata(libbuild.REPO_ROOT)


def call(cmd, stdin=None, cwd=REPO_ROOT):
    print(cmd)
    subprocess.call([expandvars(cmd)], shell=True, stdin=stdin, cwd=cwd)


# Keys:
# ./kluster.py set p gce | aws | digitalocean | linode | vultr
# ./kluster set z _zone_
# ./kluster set c _path_to_cloud_credential_

# _____ DIGITALOCEAN _____
# ./hack/kluster.py set p digitalocean
# ./hack/kluster.py set z nyc3
# ./hack/kluster.py set c conf/do.json
# ./hack/kluster.py set n 2gb:2

# _____ LINODE _____
# ./hack/kluster.py set p linode
# ./hack/kluster.py set z newark
# ./hack/kluster.py set c conf/linode.json
# ./hack/kluster.py set n 2:2
def set(k, v):
    k = k.lower()
    expand = {
        'p': 'provider',
        'z': 'zone',
        'c': 'cloud_credential',
        'n': 'nodes'
    }
    if k in expand:
        k = expand[k]
    if k == 'cloud_credential':
        v = os.path.abspath(v)
    if k in ['provider', 'zone', 'cloud_credential', 'nodes']:
        context = libbuild.read_json(KLUSTER_PATH)
        context[k.lower()] = v
        libbuild.write_json(context, KLUSTER_PATH)
    else:
        print('Unknown flag: ' + k)


def info():
    context = libbuild.read_json(KLUSTER_PATH)
    json.dump(context, sys.stdout, sort_keys=True, indent=2)


def version():
    json.dump(BUILD_METADATA, sys.stdout, sort_keys=True, indent=2)


def server():
    make.default()


def push():
    context = libbuild.read_json(KLUSTER_PATH)
    make.build_kubernetes_salt()
    make.build_cmd('start-kubernetes')
    bucket_prefix = 's3://' if context['provider'] == 'aws' else 'gs://'
    make.push(REPO_ROOT + '/dist', 'start-kubernetes-linux-amd64', bucket_prefix)
    make.push(REPO_ROOT + '/dist', 'kubernetes-salt.tar.gz', bucket_prefix)


def push_cmd():
    context = libbuild.read_json(KLUSTER_PATH)
    make.build_cmd('start-kubernetes')
    bucket_prefix = 's3://' if context['provider'] == 'aws' else 'gs://'
    make.push(REPO_ROOT + '/dist', 'start-kubernetes-linux-amd64', bucket_prefix)


def create(name='k' + datetime.datetime.fromtimestamp(time.time()).strftime('-%d%H%M')):
    context = libbuild.read_json(KLUSTER_PATH)
    context['name'] = name
    libbuild.write_json(context, KLUSTER_PATH)

    nodes = context.get('nodes', '')
    if not nodes and context['provider'] == 'aws':
        nodes = 't2.small:2'
    elif not nodes and context['provider'] == 'gce':
        nodes = 'n1-standard-2:3'
    elif context['provider'] in ['digitalocean', 'linode', 'vultr', 'scaleway', 'packet', 'softlayer']:
        pass
    else:
        print("Set cloud provider to either aws | gce | digitalocean | linode | vultr")

    cmd = "appctl cluster create --provider={provider} --nodes={nodes} --zone={zone} --version=1.5.2 {name}".format(
        provider=context['provider'],
        nodes=nodes,
        zone=context['zone'],
        version=BUILD_METADATA['version'],
        name=name)
    with open(context['cloud_credential'], 'r') as cred:
        call(cmd, stdin=cred)


# appctl cluster ssh -c cheap-cluster-20160704-1946 -i cheap-cluster-20160704-1946-master -n appscode
def ssh():
    context = libbuild.read_json(KLUSTER_PATH)
    call('appctl cluster ssh -c {0} -i {0}-master -n appscode'.format(context['name']))


def delete():
    context = libbuild.read_json(KLUSTER_PATH)
    call('appctl cluster delete {}'.format(context['name']))
    context.pop('name', '')
    libbuild.write_json(context, KLUSTER_PATH)


if __name__ == "__main__":
    if len(sys.argv) > 1:
        # http://stackoverflow.com/a/834451
        # http://stackoverflow.com/a/817296
        globals()[sys.argv[1]](*sys.argv[2:])
    else:
        print('.......................................')
        print('Try ./kluster.py (set|create|delete)')
