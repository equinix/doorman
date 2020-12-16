#!/bin/bash

source /etc/doorman-env

/bin/doormanc ${CONNECT_PUBLIC_HOSTNAME:+-f "$FACILITY"} auth --user "$common_name" --ip "$untrusted_ip" --creds "$1"
