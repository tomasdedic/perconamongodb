#!/bin/sh

SSL_DIR=/etc/psmdb-x509-tester

if [ -d "${SSL_DIR}" ]; then
	cat "${SSL_DIR}/tls.key" "${SSL_DIR}/tls.crt" >"/tmp/tls.pem"
fi

exec /main
