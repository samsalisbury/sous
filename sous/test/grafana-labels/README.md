Docker Grafana
==============

This project puts grafana on mesos via singularity/docker.

Things to note:

* This is using the bash announce wrapper.
* mini-httpd was used as it can take a port dynamically at the command line.
* Note the .docker-repo file. It defines what image is deployed.
* This can be deployed with the following: `otpl-deploy pp-uswest2 latest`
* The name announced can be found in the last line of the dockerfile. In this case, grafana.
