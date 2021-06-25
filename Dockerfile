FROM scratch
ENTRYPOINT ["/bough"]
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY bough /
