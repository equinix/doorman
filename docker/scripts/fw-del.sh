#!/usr/bin/env bash

set -eu

action=$1
vip=$2 # vpn ip

case $action in
disable)
	iptables-save | grep -vw "$vip" | iptables-restore
	;;
delete)
	ipset destroy "doorman-$vip"
	;;
*)
	echo "unknown action: $action" >&2
	exit 1
	;;
esac
