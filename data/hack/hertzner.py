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
    data = read_json('./hz.json')
    # print data
    t = []
    for d in data:
        s = d['product']
        ins = OrderedDict([
            ("external_sku", s['id']),
            ("description", s['name']),
        ])
        if s['name'].startswith('vServer'):
            ins["category"] = "Monthly Virtual Server"
        else:
            ins["category"] = "Monthly Dedicated Server"

        cpu = s['description'][0]
        if cpu.endswith(' Core CPU'):
            ins['cpu'] = int(cpu[:len(cpu)-len(' Core CPU')])
        elif 'Quad' in cpu:
            ins['cpu'] = 4
        elif 'Hexa' in cpu:
            ins['cpu'] = 6

        ram = s['description'][1]
        if ram.endswith(' GB RAM'):
            ins['ram'] = int(ram[:len(ram)-len(' GB RAM')])
        elif ram.endswith(' GB DDR3 RAM'):
            ins['ram'] = int(ram[:len(ram) - len(' GB DDR3 RAM')])
        elif ram.endswith(' GB DDR4 RAM'):
            ins['ram'] = int(ram[:len(ram) - len(' GB DDR4 RAM')])
        elif ram.endswith(' GB DDR4 ECC RAM'):
            ins['ram'] = int(ram[:len(ram) - len(' GB DDR4 ECC RAM')])

        if s['description'][2].endswith(' GB SSD'):
            ins['disk'] = int(s['description'][2][:len(s['description'][2])-len(' GB SSD')])

        t.append(ins)

    with open('/tmp/hz.json', 'w') as f:
        data = json.dumps(t, indent=2, separators=(',', ': '), ensure_ascii=False)
        f.write(data)

if __name__ == "__main__":
    sls()
