#!/usr/bin/env bash

set -v

# ensure clean slate
iptables -t filter -F DOORMAN_FORWARD
iptables -t filter -D FORWARD -j DOORMAN_FORWARD
iptables -t filter -X DOORMAN_FORWARD
iptables -t nat -F DOORMAN_POSTROUTING
iptables -t nat -D POSTROUTING -j DOORMAN_POSTROUTING
iptables -t nat -X DOORMAN_POSTROUTING

set -e
# setup chains and filters
iptables -t filter -P FORWARD DROP
iptables -t filter -N DOORMAN_FORWARD
iptables -t filter -I FORWARD -j DOORMAN_FORWARD
iptables -t nat -N DOORMAN_POSTROUTING
iptables -t nat -I POSTROUTING -j DOORMAN_POSTROUTING

# reset all doorman ipsets
ipset list -n | grep doorman- | while read -r set; do
	ipset destroy "$set"
done
