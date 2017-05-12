FROM ubuntu:latest

RUN apt-get -y update && apt-get -y install python-minimal
ADD succeed.py /usr/local/bin/succeed.py
CMD /usr/local/bin/succeed.py

LABEL \
  com.opentable.sous.repo_url=github.com/user/succeedthenfail \
  com.opentable.sous.repo_offset= \
  com.opentable.sous.version=1.0.0-succeed \
  com.opentable.sous.revision=91495f1b1630084e301241100ecf2e775f6b672c
