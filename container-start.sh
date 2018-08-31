#!/bin/sh

mkdir -p /etc/choria

cat <<EOF > /etc/choria/client.cfg
loglevel = warn
plugin.choria.middleware_hosts = ${BROKER}
EOF

cat <<EOF > /myapp.yaml
interval: 2
name: ${NAME}

# Standard Backplane specific configuration here
management:
    name: ${NAME}
    loglevel: info

    auth:
        insecure: true

    brokers:
        - ${BROKER}
EOF

if [ "${EXAMPLE}0" -eq "10" ];
then
  cd /
  exec /backplane-example $@
else
  exec /backplane $@
fi