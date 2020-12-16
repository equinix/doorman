# Install Tooling

Doorman leverages the Nix package manager to handle its dependencies for the project.
With that said, the following is the bare minimum needed tooling required on your machine to build Doorman:

* [make](https://www.gnu.org/software/make/#:~:text=GNU%20Make%20is%20a%20tool,compute%20it%20from%20other%20files.).
* [Go](https://golang.org/doc/install) 1.12 or newer.
* [Docker](https://docs.docker.com/engine/install/) or some other more sane container system.
* [git](https://git-scm.com/) to get software.
* [protobuf-compiler](https://grpc.io/docs/protoc-installation/) for generating protocol buffers.
* [docker-compose](https://docs.docker.com/compose/install/) for quick start.
* [gpg2](https://linux.die.net/man/1/gpg2) to validate signatures for Nix if used.

# Install Nix Package manager
The [Nix Package manager](https://nixos.org/learn.html) is one of the tools that is being used to handle dependencies for this project.
Nix 2.3.7 specifically at the time of this writing is the version that is being used.
To install this version of nix open up a command prompt and run the following:

```bash
  # Download the install script
curl -o install-nix-2.3.7 https://releases.nixos.org/nix/nix-2.3.7/install
  # Download the signature file
curl -o install-nix-2.3.7.asc https://releases.nixos.org/nix/nix-2.3.7/install.asc
  # Add the GPG public key to your local key chain
gpg2 --recv-keys B541D55301270E0BCF15CA5D8170B4726D7198DE
  # Verify the signature was produced by the above public key
gpg2 --verify ./install-nix-2.3.7.asc
  # Do the install of the nix package manager
sh ./install-nix-2.3.7
```

# Enter your dev enviorment
In the root of this directory there is a shell.nix file.
This file contains a list of versioned software dependencies that need to be installed for this project.
Assuming you have installed the Nix package manager from the previous step you can run the [nix-shell](https://nixos.org/nix/manual/#sec-nix-shell) command which will create a sandbox environment.
Like all other shells, when you are done with your work you can type `exit` to return to your normal shell.

# Setting up an End to End test environment
Other environments can be used such as VMWare Fusion, VirtualBox, etc depending on your usecase.
You can add those providers to the Vagrantfile within the repo if you could like to do that.
The initial thought was to have a machine provisioned for development and isolation.

### Ubuntu 20.04
This setup assumes you have access to a machine, either local or in the cloud and that you have root access.

#### Install Hypervisor and other tooling
As root:
```bash
apt-get update
apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virtinst virt-manager ebtables dnsmasq-base build-essential vagrant ruby-libvirt libxslt-dev libxml2-dev libvirt-dev zlib1g-dev ruby-dev
systemctl is-active libvirtd
usermod -aG libvirt $USER
usermod -aG kvm $USER
vagrant plugin install vagrant-libvirt
```

##### Now head on over to the: end to end testing section.
