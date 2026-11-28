FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends samba samba-ad-dc samba-ad-provision winbind krb5-user ldb-tools && \
    rm -rf /var/lib/apt/lists/*
COPY start.sh /start.sh
ENTRYPOINT ["bash", "/start.sh"]
