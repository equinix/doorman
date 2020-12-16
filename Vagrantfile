# -*- mode: ruby -*-
# vi: set ft=ruby :

# All Vagrant configuration is done below. The "2" in Vagrant.configure
# configures the configuration version (we support older styles for
# backwards compatibility). Please don't change it unless you know what
# you're doing.
Vagrant.configure("2") do |config|

  # This Vagrant file contains an end to end test case for Doorman.
  # Create three instances:
  # A client instance that shares a network cidr with the server instance.
  # A server instance that shares a network cidr with both the client instance and a target instance.
  # A target instance that shares a network cidr with the server instance.
  #
  # All three instances will be brought up.
  # Doorman and its dependencies will be installed on the server instance.
  # OpenVPN configuration file will be moved from the server instance to the client instance.
  # The client instance will establish an OpenVPN connection to the server instance.
  # The client instance will ping the target machine.

  # This instance will be the client.
  # It will share a "public" network with the server instance.
  # The goal of this instance is to connect to the target instance.
  config.vm.define "client" do |client|
    client.vm.box = "generic/ubuntu2004"
    client.vm.box_version = "3.0.28"
    client.vm.synced_folder "./e2e", "/vagrant"
    # Install OpenVPN
    # Create a credential file that contains known invalid credentials. (The external calls to the Equinix API have fake data returned)
    client.vm.provision "shell", inline: <<-SHELL
        apt-get update
        apt-get install -y openvpn
        touch /vagrant/creds.txt
        echo 00000000-0000-0000-0000-000000000001 >> /vagrant/creds.txt
        echo password >> /vagrant/creds.txt
        chmod 777 -R /vagrant
    SHELL
    client.vm.network "private_network", ip: "10.88.110.10" # Client network
    client.vm.provider "libvirt" do |kvm|
      kvm.memory = "512"
      kvm.cpus = 1
    end
  end

  # This instance will be the server.
  # It will share a "public" network with the client instance, and share a "private" network with the target instance.
  # This is the instance that will run the Doorman software, and well be the instance to log into for debugging purposes.
  config.vm.define "server" do |server|
    server.vm.box = "generic/ubuntu2004"
    server.vm.box_version = "3.0.28"
    # Put the compiled binaries and scripts into the VM
    server.vm.provision "file", source: "./docker", destination: "/home/vagrant/doorman/docker"
    server.vm.provision "file", source: "./cmd", destination: "/home/vagrant/doorman/cmd"
    server.vm.provision "file", source: "./docker-compose.yml", destination: "/home/vagrant/doorman/docker-compose.yml"

    # Set environment to testing.
    # Change the magic ip address.
    # Change api url so no actual APIs are touched during testing.
    server.vm.provision "shell", inline: <<-SHELL
            sed -i 's/production/testing/g' /home/vagrant/doorman/docker-compose.yml
            sed -i 's,https://api-internal.equinix.com,http://localhost,g' /home/vagrant/doorman/docker-compose.yml
    SHELL

    # Install all of the software dependencies
    server.vm.provision "shell", inline: <<-SHELL
        apt-get update
        apt-get install -y docker.io docker-compose sshpass
        systemctl enable --now docker
        usermod -aG docker vagrant
    SHELL

    # Start Doorman Service
    server.vm.provision "shell", inline: <<-SHELL
        cd /home/vagrant/doorman
        docker-compose up -d
        echo '--> WAITING FOR OPENVPN PORT TO BE OPEN <--'
        docker-compose exec -T server sh -c 'until nc -w 1 localhost 1194 </dev/null; do sleep 1; done'
        echo '--> WAITING FOR DOORMAN PORT TO BE OPEN <--'
        docker-compose exec -T server sh -c 'until nc -w 1 localhost 8080 </dev/null; do sleep 1; done'
    SHELL

    # Enter docker container running Doorman without a TTY session, use Doormanc cli to create OpenVPN config file, and pipe the standard out to a file
    server.vm.provision "shell", inline: <<-SHELL
        cd /home/vagrant/doorman
        docker-compose exec -T server "/bin/doormanc" "create-client" "-u" "00000000-0000-0000-0000-000000000001" > vagrant-client.ovpn
    SHELL

    # Change the remote ip address in the OpenVPN config i.e. "The Magic Ip" to the server instance's IP.
    # Change authentication method, use a file.
    # Copy the complete ovpn client config to the client machine
    server.vm.provision "shell", inline: <<-SHELL
        sed -i 's/vpn.lab1.platformequinix.com/10.88.110.11/g' /home/vagrant/doorman/vagrant-client.ovpn
        sed -i 's/auth-user-pass/auth-user-pass creds.txt/g' /home/vagrant/doorman/vagrant-client.ovpn
        mkdir ~/.ssh
        ssh-keyscan 10.88.110.10 >> ~/.ssh/known_hosts
        sshpass -p "vagrant" scp /home/vagrant/doorman/vagrant-client.ovpn vagrant@10.88.110.10:/vagrant
    SHELL

    server.vm.network "private_network", ip: "10.88.110.11" # Client network
    server.vm.network "private_network", ip: "10.88.111.10" # Private network
    server.vm.provider "libvirt" do |kvm|
      kvm.memory = "512"
      kvm.cpus = 1
    end
  end

  # This instance will be the target.
  # It will share a "private" network with the server instance.
  # The goal of this instance is to receive traffic from the client instance.
  config.vm.define "target" do |target|
    target.vm.box = "generic/ubuntu2004"
    target.vm.box_version = "3.0.28"
    target.vm.network "private_network", ip: "10.88.111.11" # Private network
    target.vm.provider "libvirt" do |kvm|
      kvm.memory = "512"
      kvm.cpus = 1
    end
  end
end
