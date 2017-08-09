#!/usr/bin/env python

import json
from collections import OrderedDict


def sls():
    spec = OrderedDict([
        (1, [1, 2, 4, 6, 8, 12, 16]),
        (2, [1, 2, 4, 6, 8, 12, 16]),
        (4, [1, 2, 4, 6, 8, 12, 16, 32, 48, 64]),
        (8, [1, 2, 4, 6, 8, 12, 16, 32, 48, 64]),
        (12, [1, 2, 4, 6, 8, 12, 16, 32, 48, 64]),
        (16, [1, 2, 4, 6, 8, 12, 16, 32, 48, 64, 128]),
        (32, [1, 2, 4, 6, 8, 12, 16, 32, 48, 64, 128, 242]),
        (56, [1, 2, 4, 6, 8, 12, 16, 32, 48, 64, 128, 242])
    ])
    t = []
    for cpu, ms in spec.iteritems():
        for m in ms:
            print '{0}c{1}m'.format(cpu, m)
            ins = OrderedDict([
                ("external_sku", '{0}c{1}m'.format(cpu, m)),
                ("description", ('{} core and {}GB RAM' if cpu == 1 else '{} cores and {}GB RAM').format(cpu, m)),
                ("category", "Hourly Virtual Server"),
                ("cpu", cpu),
                ("ram", m)
            ])
            t.append(ins)
    print t
    with open('/tmp/softlayer.json', 'w') as f:
        data = json.dumps(t, indent=2, separators=(',', ': '), ensure_ascii=False)
        f.write(data)

if __name__ == "__main__":
    sls()
