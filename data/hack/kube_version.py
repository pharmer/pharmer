#!/usr/bin/env python

import io
import json
from collections import OrderedDict
from os.path import expandvars
import sys

ROOT = expandvars("$GOPATH/src/github.com/appscode/data")


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


def add():
    f = ROOT + '/files/cloud_provider.json'
    v = json.loads("""{
    "version": "1.7.0",
    "description": "1.7.0",
    "apps": {
        "kubernetes-server": "1.7.0",
        "kubernetes-salt": "1.7.0-ac",
        "start-kubernetes": "0.6.29",
        "hostfacts": "1.5.7"
    }
}
""", object_pairs_hook=OrderedDict)
    data = read_json(f)
    for provider, pdata in data['cloud_provider'].iteritems():
        for env, edata in pdata['kubernetes']['versions_by_env'].iteritems():
            if env != 'prod':
                edata.append(v)
    write_json(data, f)


def release():
    f = ROOT + '/files/cloud_provider.json'
    v = json.loads("""{
    "version": "1.5.7",
    "description": "1.5.7",
    "apps": {
        "kubernetes-server": "1.5.7",
        "kubernetes-salt": "1.5.7-ac",
        "start-kubernetes": "0.6.29",
        "hostfacts": "1.5.7"
    }
}
""", object_pairs_hook=OrderedDict)
    data = read_json(f)
    for provider, pdata in data['cloud_provider'].iteritems():
        for env, edata in pdata['kubernetes']['versions_by_env'].iteritems():
            if env == 'prod':
                edata.append(v)
    write_json(data, f)


def deprecate(version):
    f = ROOT + '/files/cloud_provider.json'
    data = read_json(f)
    for provider, pdata in data['cloud_provider'].iteritems():
        for env, edata in pdata['kubernetes']['versions_by_env'].iteritems():
            for v in edata:
                if v['version'] == version:
                    v['deprecated'] = True
    write_json(data, f)


if __name__ == "__main__":
    if len(sys.argv) > 1:
        # http://stackoverflow.com/a/834451
        # http://stackoverflow.com/a/817296
        globals()[sys.argv[1]](*sys.argv[2:])
    else:
        die('Missing command.')
