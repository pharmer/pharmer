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
        del pdata['db']
    write_json(data, f)


if __name__ == "__main__":
    release()
