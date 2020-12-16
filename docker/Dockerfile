FROM alpine:3.7
ENV EASYRSA_BATCH=yes
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/doorman", "-s"]
VOLUME /etc/openvpn/easy-rsa
EXPOSE 1194/tcp
EXPOSE 8080/tcp
WORKDIR /root

# setup
RUN \
    apk add --no-cache --update --upgrade bash ca-certificates easy-rsa ipset openvpn && \
    apk add --no-cache --update --upgrade --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing cfssl && \
    mkdir -p /etc/openvpn/ccd /var/log/doorman

COPY docker/tls /tls
COPY docker/openvpn/* /etc/openvpn/
COPY docker/entrypoint.sh /entrypoint.sh
COPY docker/scripts /app/
COPY cmd/doorman/doorman-x86_64-linux /bin/doorman
COPY cmd/doormanc/doormanc-x86_64-linux /bin/doormanc
