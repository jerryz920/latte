import json

import sys

with open(sys.argv[1], "r") as f:
    for l in f:
        val=json.loads(l)
        if val["msg"] == "finish":
            print("%f" % val["time"])
        else:
            print("%s %f" % (val["name"], val["time"]))
