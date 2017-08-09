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


def release():
    f = ROOT + '/files/cloud_provider.json'
    data = read_json(f)
    for provider, pdata in data['cloud_provider'].iteritems():
        kubes = pdata['kubernetes']['versions_by_env']
        if 'prod' in kubes and 'onebox' not in kubes:
            kubes['onebox'] = kubes['prod']

    for provider, pdata in data['cloud_provider'].iteritems():
        for dbdata in pdata['db'].values():
            dbs = dbdata['versions_by_env']
            if 'prod' in dbs and 'onebox' not in dbs:
                dbs['onebox'] = dbs['prod']

    write_json(data, f)


if __name__ == "__main__":
    release()
