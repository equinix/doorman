# Doorman

Doorman is a customer facing VPN service that gives access to their private subnets within their [Equinix Metal](https://metal.equinix.com/) environment.
It works by managing OpenVPN credentials/authorization and iptables rules.
All of the vpn tech is vanilla OpenVPN.

# Note
With the upcoming EoL of Equinix Metal on June 30, 2026, this repo is being archived on February 28, 2026.

## Documentation
Install [make](https://www.gnu.org/software/make/#:~:text=GNU%20Make%20is%20a%20tool,compute%20it%20from%20other%20files.), [Docker](https://docs.docker.com/engine/install/) and run the following:  
`make documentation`

Open a web browser and goto http://0.0.0.0:8000.

## Quickstart - Equinix Metal deployment

### Pre-requisites:
1.  Equinix Metal Account
1.  Equinix Metal User API Key
1.  SSH Key associated with your Equinix Metal Account
1.  Enable Two Factor Authentication


Provision a server in the metro of your choice with the Ubuntu 20.04 operating system. 
SSH into your newly provisioned server, clone this repository and run the `quickstart.sh` script.

Example output:
```
doorman-demo@doorman:~/oss-doorman$ ./quickstart.sh 
Would you like to setup OpenVPN and use Doorman as authentication? [y/N]: y
Please enter your API token, note this is NOT your password.
For more information see https://metal.equinix.com/developers/api/authentication/
API token: REDACTED

Credential's verified. Moving onto installing Doorman!
Installing tooling:
Hit:1 http://security.ubuntu.com/ubuntu focal-security InRelease
Hit:2 http://archive.ubuntu.com/ubuntu focal InRelease
Hit:3 http://archive.ubuntu.com/ubuntu focal-updates InRelease
Hit:4 http://archive.ubuntu.com/ubuntu focal-backports InRelease
Reading package lists... Done
Reading package lists... Done
...
...
Creating login credentials for user:
Your OpenVPN configuration file can be found at /home/doorman-demo/oss-doorman/doorman-demo@equinix.com.ovpn
doorman-demo@doorman:~/oss-doorman$
```

OpenVPN and Doorman should be running now on your provisioned server. 
The contents of the OpenVPN configuration file will be what you use with your OpenVPN client. 
The username will be the email address you use to login to your Equinix Metal account. 
Your password will be the current six digit two factor authentication token prepended to your Equinix Metal password.

Password Example:
```
123456P@ssw0rd
```

To access the private IP addresses of instances in other facilities you must enable "Backend Transfer", which you'll find under the "IP & Networking" menu of the Equinix Metal Console for your project.
