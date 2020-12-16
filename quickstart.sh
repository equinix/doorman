#!/bin/bash

read -p "Would you like to setup OpenVPN and use Doorman as authentication? [y/N]: " -n 1 -r
echo    # (optional) move to a new line
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo "Exiting!"
    [[ "$0" = "$BASH_SOURCE" ]] && exit 1 || return 1 # handle exits from shell or function but don't exit interactive shell
fi

echo "Please enter your API token, note this is NOT your password."
echo "For more information see https://metal.equinix.com/developers/api/authentication/"
read -p "API token: " -r -s
if [[ -z "$REPLY" ]]
then
    echo "No API Token entered, exiting."
    [[ "$0" = "$BASH_SOURCE" ]] && exit 1 || return 1 # handle exits from shell or function but don't exit interactive shell
else
    API_TOKEN=$REPLY
fi

echo    # (optional) move to a new line
AUTHENTICATION_CALL="/usr/bin/curl -s -H 'X-Auth-Token: $API_TOKEN' https://api.equinix.com/metal/v1/user"
USER_DETAILS=$(eval $AUTHENTICATION_CALL)
IS_TOKEN_BAD=$(echo $USER_DETAILS | jq 'select(.error != null)')
if [[ ! -z "$IS_TOKEN_BAD" ]]
then
    echo "Invalid Token"
    [[ "$0" = "$BASH_SOURCE" ]] && exit 1 || return 1 # handle exits from shell or function but don't exit interactive shell
fi
USER_ID=$(echo $USER_DETAILS | jq -r .id)
USER_EMAIL=$(echo $USER_DETAILS | jq -r .email)
echo "Credential's verified. Moving onto installing Doorman!"
echo "Installing tooling:"
apt-get update
apt-get install -y golang git docker.io protobuf-compiler haveged
curl -L "https://github.com/docker/compose/releases/download/1.26.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
echo "Building Doorman plugin:"
make
echo "Getting public ipv4 address:"
PUBLIC_IP=$(eval ip addr show bond0 | grep 'inet ' | awk '{print $2}' | cut -f1 -d'/' | head -n 1)
echo "Setting configuration to use public ipv4p address"
sed -i 's,https://api-internal.equinix.com,"https://api.equinix.com/metal/v1",g' ./docker-compose.yml
sed -i "s,127.0.0.1,$PUBLIC_IP,g" ./docker-compose.yml
echo "Starting OpenVPN with Doorman authentication plugin:"
docker-compose up -d
echo '--> WAITING FOR OPENVPN PORT TO BE OPEN <--'
docker-compose exec -T server sh -c 'until nc -w 1 localhost 1194 </dev/null; do sleep 1; done'
echo '--> WAITING FOR DOORMAN PORT TO BE OPEN <--'
docker-compose exec -T server sh -c 'until nc -w 1 localhost 8080 </dev/null; do sleep 1; done'
echo "Creating login credentials for user:"
docker-compose exec -T server "/bin/doormanc" "create-client" "-u" "$USER_ID" > $USER_EMAIL.ovpn
sed -i "s/vpn.lab1.platformequinix.com/$PUBLIC_IP/g" `pwd`/$USER_EMAIL.ovpn
echo "Your OpenVPN configuration file can be found at `pwd`/$USER_EMAIL.ovpn"
