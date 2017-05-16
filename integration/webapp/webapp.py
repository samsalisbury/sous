#!/usr/bin/env python

import os
from bottle import Bottle, run, response

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

mesos_port = os.environ["PORT0"]
mesos_host = os.environ["TASK_HOST"]
run(app, host=mesos_host, port=mesos_port)
