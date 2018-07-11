#!/bin/bash

mkdir -p /etc/puppetlabs/mcollective/

cat <<EOF > /etc/puppetlabs/mcollective/client.cfg
loglevel = warn
plugin.choria.middleware_hosts = ${BROKER}
EOF

cat <<EOF > /myapp.yaml
# your own config here
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