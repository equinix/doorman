#!/usr/bin/env bash

set -euo pipefail

usage() {
	echo "usage: $0 [--ignore-errors] --client=client-uuid" >&2
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
	--ignore-errors) echo "ignoring errors" >&2 && set +e ;;
	-h | --help) usage && exit 0 ;;
	*) usage && exit 1 ;;
	esac
done
[[ -z "${client:-}" ]] && echo "--client arg is required" >&2 && usage && exit 1

cd /etc/openvpn/easy-rsa

# If file exits revoke it, otherwise log that nothing is happening
if [[ -f "/etc/openvpn/easy-rsa/pki/issued/$client.crt"  ]]; then
/usr/share/easy-rsa/easyrsa revoke "$client"
else
  echo "no client file found.  nothing to revoke"
  exit 0
fi
cd pki
find . -type f -name "$client.*" -delete
