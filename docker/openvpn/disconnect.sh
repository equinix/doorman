#!/bin/bash

source /etc/doorman-env

/bin/doormanc ${CONNECT_PUBLIC_HOSTNAME:+-f "$FACILITY"} disconnect --user "${common_name}"
