#!/usr/bin/env python

import json, sys
from collections import OrderedDict


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
    data = read_json('/Users/tamal/Desktop/ali.json')
    # print data
    t = {}
    for d in data:
        sku = d['external_sku']
        if sku not in t:
            t[sku] = d
        else:
            t[sku]['regions'].append(d['regions'][0])

    for sku, d in t.iteritems():
        d['regions'] = sorted(set(d['regions']))

    a = []
    for sku in sorted(t.keys()):
        a.append(t[sku])

    with open('/Users/tamal/Desktop/alii.json', 'w') as f:
        data = json.dumps(a, indent=2, separators=(',', ': '), ensure_ascii=False)
        f.write(data)

if __name__ == "__main__":
    sls()
