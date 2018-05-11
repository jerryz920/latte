#!/usr/bin/python3
import numpy
import json

import argparse

def conf():

    parser = argparse.ArgumentParser()
    parser.add_argument("-r", "--records", type=int, nargs="+", dest="records")
    parser.add_argument("-o", "--op", type=str, nargs="?", dest="op")
    parser.add_argument("-m", "--member", type=int, nargs="?", dest="nmember", default=10)
    parser.add_argument("-c", "--check", type=str, nargs="?", dest="check", default="any")

    return parser.parse_args()

if __name__ == "__main__":

    c = conf()
    
    outname = "latency-group-%d-%s-%s" % (c.nmember, c.check,  c.op)

    with open(outname, "w") as fout:
        for r in c.records:
            fname = "results/glog-%d" % r
            with open(fname, "r") as fin:
                for l in fin:
                    fdata = json.loads(l)
                    if "name" in fdata:
                        if fdata["name"][1:] == c.op and (fdata["speaker"].endswith(c.check) or
                                c.check == "any"):
                            fout.write(str(fdata["time"]) + "\n")
