#!/usr/bin/env python

import io
import json
from collections import OrderedDict
from os.path import expandvars
import sys
import urllib

REPO_ROOT= '/home/tamal/go/src/github.com/appscode/data'

def die(status):
    if status:
        sys.exit(status)


# TODO: use unicode encoding
def read_json(name):
    try:
        with open(name, 'r') as f:
            return json.load(f, object_pairs_hook=OrderedDict)
    except IOError:
        return {}


def write_json(obj, name):
    with io.open(name, 'w') as f:
        data = json.dumps(obj, indent=2, separators=(',', ': '), ensure_ascii=False)
        f.write(data)


def sls():
    clouds = read_json(REPO_ROOT + '/files/cloud_provider.json')

    response = urllib.urlopen('https://api.vultr.com/v1/plans/list')
    vultr = json.loads(response.read())

    # Fix deprecated tag for existing ones
    vms = clouds['cloud_provider']['vultr']['instance_types']
    for vm in vms:
        if vultr[vm['external_sku']].get('deprecated', False):
            vm['deprecated'] = True
        elif 'deprecated' in vm:
            del vm['deprecated']
        del vultr[vm['external_sku']]

    # add new ones
    for k, plan in vultr.items():
        if plan.get('deprecated', False):
            continue
        print plan['VPSPLANID']

        ram = int(int(plan['ram'])/1024)
        if ram:
            vms.append(OrderedDict([
                ('external_sku', plan['VPSPLANID']),
                ('description', plan['name']),
                ('category', plan['plan_type']),
                ('cpu', plan['vcpu_count']),
                ('ram', ram),
                ('disk', plan['disk']),
                ]))

    write_json(clouds, REPO_ROOT + '/files/cloud_provider.json')


if __name__ == "__main__":
    sls()
