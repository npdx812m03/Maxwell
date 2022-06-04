import numpy as np
import os

s = os.walk("data/fib10")
for r, d, f in s:
    for ff in f:
        if "1.txt" in ff:
            fff = open(r + ".log", "a")
            for line in open(r + "/" + ff, "r").readlines():
                fff.write(line)
            fff.close()

