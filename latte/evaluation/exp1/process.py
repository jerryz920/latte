import json

import sys
if len(sys.argv) >= 2:
    with open(sys.argv[1], "r") as f:
        for l in f:
            try:
                val=json.loads(l)
            except:
                continue
            if not "time" in val:
                continue
            if val["msg"] == "finish":
                print("%f" % val["time"])
            else:
                print("%s %f" % (val["name"], val["time"]))

else:
    for line in sys.stdin:
        try:
            val=json.loads(line)
        except:
            continue
        if not "time" in val:
            continue

        if val["msg"] == "finish":
            print("%f" % val["time"])
        else:
            print("%s %f" % (val["name"], val["time"]))
