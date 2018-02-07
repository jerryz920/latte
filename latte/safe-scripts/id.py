#!/usr/bin/python


import json
#
#{
# "message": "['okPL5d4PvUhoGngdyuK4koBPfWhvuoUX9Ad4Ym5k-p8']"
#}
#

import sys

if __name__ == "__main__":
    s = json.load(sys.stdin)
    data = eval(s["message"])
    print(data[0])




