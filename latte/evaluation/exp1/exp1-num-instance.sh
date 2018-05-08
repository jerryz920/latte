#!/bin/bash

#Thread = 1
#N VM * 50 Ctn * 4 Proc
#N = 200, 400, 600, 800, 1000
#Ops = create, config (5 for vm, 20 for ctn, 20 for proc), delete, fetch (simplest guard)
#Order1: create config fetch | delete (see if fetch is impacted)
#Order2: random (create config fetch delete)
#Measure: 
#Order 1 throughput, tail latency of N=1000
#Order 2 throughput, tail latency of N=1000
#





