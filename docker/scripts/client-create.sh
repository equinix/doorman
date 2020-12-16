#!/usr/bin/env bash

set -euo pipefail

usage() {
	echo "usage: $0 --client=client-uuid" >&2
}

for arg; do
	case "$arg" in
	--client=*)
		client=${arg#*=}
		hex='[[:xdigit:]]'
		if ! [[ "$client" =~ ^$hex{8}-$hex{4}-$hex{4}-$hex{4}-$hex{12}$ ]]; then
			echo "--client arg is not in required uuid format" >&2
			exit 1
		fi
		;;
	-h | --help) usage && exit 0 ;;
	*) usage && exit 1 ;;
	esac
done
[[ -z "${client:-}" ]] && echo "--client arg is required" >&2 && usage && exit 1

cd /etc/openvpn/easy-rsa
/usr/share/easy-rsa/easyrsa build-client-full "$client" nopass
