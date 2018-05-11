#!/usr/bin/python3
import numpy
import json

import argparse

def conf():

    parser = argparse.ArgumentParser()
    parser.add_argument("-r", "--records", type=int, nargs="+", dest="records")
    parser.add_argument("-m", "--mode", type=int, nargs="?", dest="mode")
    parser.add_argument("-o", "--op", type=str, nargs="?", dest="op")
    parser.add_argument("-t", "--thread", type=int, nargs="?", dest="nthread")
    parser.add_argument("-n", "--instance", type=int, nargs="?", dest="ninstance", default=1024)
    parser.add_argument("-l", "--level", type=int, nargs="?", dest="nlevel", default=3)

    return parser.parse_args()

if __name__ == "__main__":

    c = conf()
    
    if c.mode == 1:
        outname = "latency-%d-seq-%d-%d" % (c.nthread, c.ninstance, c.nlevel)
        opout = "latency-%d-%s-seq-%d-%d" % (c.nthread, c.op, c.ninstance, c.nlevel)
    else:
        outname = "latency-%d-blk-%d-%d" % (c.nthread, c.ninstance, c.nlevel)
        opout = "latency-%d-%s-blk-%d-%d" % (c.nthread, c.op, c.ninstance, c.nlevel)

    with open(outname, "w") as fout:
        with open(opout, "w") as  fopout:
            for r in c.records:
                if (c.mode == 1 and int(r) % 2 == 0) or \
                        (c.mode == 0 and int(r) % 2 == 1):
                    fname = "results/perf-log-%s" % r
                    with open(fname, "r") as fin:
                        for l in fin:
                            fdata = json.loads(l)
                            if "name" in fdata:
                                fout.write(str(fdata["time"]) + "\n")
                                if fdata["name"][1:] == c.op:
                                    fopout.write(str(fdata["time"]) + "\n")
