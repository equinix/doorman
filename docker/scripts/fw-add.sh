#!/usr/bin/env bash

set -eu

do_undo() {
	iptables-save | grep -vw "$vip" | iptables-restore
}

action=$1
vip=$2 # vpn ip

case $action in
create)
	if ipset -q list "doorman-$vip"; then
		echo "doorman-$vip already exists!" >&2
		exit 1
	fi
	ipset create "doorman-$vip" hash:net,net
	;;
add)
	cnet=$3 # client private subnet
	ipset add "doorman-$vip" "$vip,$cnet"
	ipset add "doorman-$vip" "$cnet,$vip"
	;;
enable)
	if iptables-save | grep -qw "$vip"; then
		echo "$vip already exists in iptables!" >&2
		iptables-save | grep -E '(^\*|\b'"$vip"'\b)' >&2
		exit 1
	fi

	trap do_undo ERR

	magic=${3:-} # magic ip
	iptables -t filter -A DOORMAN_FORWARD -m set --match-set "doorman-$vip" src,dst -j ACCEPT
	if [[ -z ${magic:-} ]]; then
		iptables -t nat -I DOORMAN_POSTROUTING 1 -m set --match-set "doorman-$vip" src,dst -s "$vip" -j MASQUERADE
	else
		iptables -t nat -I DOORMAN_POSTROUTING 1 -m set --match-set "doorman-$vip" src,dst -s "$vip" -j SNAT --to-source "$magic"
	fi
	;;
*)
	echo "unknown action: $action" >&2
	exit 1
	;;
esac
