#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail -o xtrace

cmd=$(basename "$1")

if [[ $cmd != doorman ]] && [[ $cmd != doormanc ]]; then
	exec "$@"
fi

if ! [[ -d /etc/openvpn/easy-rsa/pki ]]; then
	cd /etc/openvpn/easy-rsa
	echo "generating openvpn pki stuffs, please hold"
	/usr/share/easy-rsa/easyrsa init-pki
	/usr/share/easy-rsa/easyrsa \
		--req-c=US \
		--req-st="California" \
		--req-city="Redwood City" \
		--req-org="Equinix Metal" \
		--req-email="support@equinix.com" \
		--req-ou="Equinix Metal Operations" \
		build-ca nopass
	/usr/share/easy-rsa/easyrsa --req-cn="vpn.equinix.com" gen-req server nopass
	/usr/share/easy-rsa/easyrsa sign-req server server
	/usr/share/easy-rsa/easyrsa gen-dh
	cd "$OLDPWD"
fi

if [[ -z ${GRPC_INSECURE:-} ]]; then
	if [[ $cmd == doorman ]]; then
		if ! [[ -r /etc/openvpn/doorman-grpc/server-key.pem ]]; then
			(
				cd /etc/openvpn/doorman-grpc
				FACILITY=$(echo "$FACILITY" | tr '[:upper:]' '[:lower:]') sh /tls/gencerts.sh
				ln -nsf bundle.pem "server-$FACILITY.crt"
				ln -nsf server-key.pem "server-$FACILITY.key"
			)
		fi
		set +x
		# we don't want to show the tls key
		GRPC_KEY=$(cat /etc/openvpn/doorman-grpc/server-key.pem)
		set -x
		GRPC_CERT=$(cat /etc/openvpn/doorman-grpc/bundle.pem)
		export GRPC_KEY
		export GRPC_CERT
	elif [[ -z ${GRPC_CERT:-} ]]; then
		GRPC_CERT=$(cat /etc/openvpn/doorman-grpc/bundle.pem)
		export GRPC_CERT
	fi
fi

cat >/etc/doorman-env <<EOF
export CONNECT_PUBLIC_HOSTNAME=${GRPC_CERT:+true}
export FACILITY=$FACILITY
export GRPC_CERT="${GRPC_CERT:-}"
export GRPC_INSECURE="${GRPC_INSECURE:-}"
export EQUINIX_ENV=$EQUINIX_ENV
export EQUINIX_VERSION=$EQUINIX_VERSION
EOF

"$@"
