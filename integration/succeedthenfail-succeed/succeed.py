#!/usr/bin/env python

import signal, time

def handler(signum, frame):
    print("exit on signal:{}".format(signum))
    exit(0)

def main():
    signal.signal(signal.SIGTERM, handler)
    while True:
        print("happily succeeding.")
        time.sleep(10)

main()
