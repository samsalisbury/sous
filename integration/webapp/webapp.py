#!/usr/bin/env python

import os
import signal
import time
from bottle import Bottle, run, response

BECOME_HEALTHY_SECONDS=60

start_time = time.time()
app = Bottle()

@app.route('/healthy')
def healthy():
    print("hit healthcheck route")
    return("healthy")

@app.route('/')
def root():
    print("hit webroot")
    return("root")

@app.route('/sick')
def sick():
    print("hit sick route.")
    response.status = 400
    return("sick")

@app.route('/slowhealthy')
def slowhealthy():
    print("hit slow-healthy route.")
    now = time.time()
    elapsed = int(now - start_time)
    if elapsed <= BECOME_HEALTHY_SECONDS:
        response.status = 400
        return "Still ill. Running time:{}s. Eventually healthy at:{}s".format(
            elapsed, BECOME_HEALTHY_SECONDS)
    return("Became healthy at {}s, {}s ago.".format(
            BECOME_HEALTHY_SECONDS, (elapsed - BECOME_HEALTHY_SECONDS)))

def handler(signum, frame):
    print("exit on signal:{}".format(signum))
    exit(0)

signal.signal(signal.SIGTERM, handler)
mesos_port = os.environ["PORT0"]
mesos_host = os.environ["TASK_HOST"]
run(app, host=mesos_host, port=mesos_port)
