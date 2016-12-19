FROM alpine:3.4
COPY git_pubkey_rsa.pub /root/.ssh/authorized_keys
ADD repos.tgz /
# ADD sshd_config /etc/ssh/sshd_config
RUN \
  apk update && \
  apk add openssh git && \
  mkdir -p /var/empty && \
  chmod -R go-rwx /root/.ssh && \
  ssh-keygen -A
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
