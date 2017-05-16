FROM alpine:latest

RUN apk update && apk add python py-pip
RUN mkdir -p /srv/webapp
ADD webapp.py /srv/webapp/webapp.py
ADD requirements.txt /srv/webapp/requirements.txt
RUN pip install -r /srv/webapp/requirements.txt
CMD python /srv/webapp/webapp.py
