# End to End testing

### Production Usecase
In production Doorman is leveraging special nice to haves which makes it fast and reliable but difficult to test without disrupting production.
For example, as described in the architecture section in greater detail:

1. The equinix metal networking uses [ACLs](https://en.wikipedia.org/wiki/Access-control_list) and other network magic to expose an IP address to the server that exposes Doorman to the internet.
1. When a user wishes to use Doorman, the user leverages the customer facing UI to generate a new client [OpenVPN profile](https://openvpn.net/faq/what-is-a-client-configuration-or-connection-profile/).
1. Once the user downloads the OpenVPN client profile and attempts to connect, the user is required to pass a username and a password which has a [mfa token](https://en.wikipedia.org/wiki/Multi-factor_authentication) prefixed.
1. Doorman communicates with the production API to validate the credentials and if sucessful returns a list of [private subnets](https://en.wikipedia.org/wiki/Private_network) a user has access too.
1. Doorman then uses this list of subnets to build routes with OpenVPN for the user attempting to connect.
1. Once the routes have been established Doorman keeps a connection open with the OpenVPN client and proxies all traffic to and from Private subnets within Equinix Metal.

### Challenges for End to End Tests
The above introduces several challenges:
1. How can we test for an application that expects networking magic?
1. How can we test this process without having to interact with the production API?
1. How can we simulate the networking for 3 machines on two different networks?

### Proposed solution
The imperfect solution I have chosen is using [Virtual Machines](https://en.wikipedia.org/wiki/Virtual_machine) on an open source operating system.
Specifically using [vagrant](https://www.vagrantup.com/) which is used to drive [libvirt](https://libvirt.org/).

*Why?! This is the age of containers!!!*  You are correct of course, this is one approach of many that can and maybe should have been used.
I chose this solution because:

* Vagrant is relatively easy to install, operate, and can configure networks and virtual machines with a single configuration file.
* We can make the virtual environment small enough so that you can close a tab in chrome ant have enough resources to run it!
* It allows for the user to truly have an isolated environment for their testing.

Also keep in mind, pull requests for this repository are open, so if you would like another workflow to deprecate this one, open a PR!

### How to use

The next several steps assume you have followed the tooling guide and have installed all of the requisite software.
This guide also assumes you have cloned the [Doorman](https://github.com/equinix/doorman) git repository onto that machine so you have access to the source code.
All of the commands in the next three steps are expected to be run from the root directory of this repository.

#### Run unit tests
Test basic functionality with unit tests, if these don't work there is no sense in building a test environment as this will provide you with the quickest feedback.
```bash
 make test
```

#### Build Binaries
Build both the doorman and doormanc binaries so that they can be leveraged when standing up a test environment.
```bash
make
```

#### Build a test environment
Once you can confirm that your unit tests pass and Doorman builds cleanly, you can stand up your development environment.

```bash
make e2e
```

You will see alot of scroll by but this is what is happening:

1. Three machines are being created, a client, a server, and a target.
1. Two networks are being created, a client network and a server network.
   The server has access to both networks while the client and target [instances](https://cloud.google.com/compute/docs/instances/).
1. Install the OpenVPN Client on the client instance.
1. Run doorman on the server as it would be in a production environment.
1. Use doormanc to create an OpenVPN client profile (or configueration file) for the client machine.
1. Edit the OpenVPN client profile to that it will attempt to connect to the doorman server.
1. Place this edited OpenVPN clint profile in a location where it can accessed by the client instance.

#### Running a manual test

1. Validate that you cannot ping the target instance from the client instance.
    ```bash
    vagrant ssh client -c 'ping -c 1 10.88.111.11'
    ```
   Example output to confirm lack of connectivity:
    ```bash
    PING 10.88.111.11 (10.88.111.11) 56(84) bytes of data.
    From 192.168.121.1 icmp_seq=1 Destination Port Unreachable

    --- 10.88.111.11 ping statistics ---
    1 packets transmitted, 0 received, +1 errors, 100% packet loss, time 0ms

   Connection to 192.168.121.57 closed.
    ```

1. Login to the server instance that is running Doorman and get the openvpn config file. 
    ```bash
    vagrant ssh server
    ```
   Copy the output of the openvpn file into `/vagrant/vagrant-client.ovpn`.
1. Login to the client instance and create an OpenVPN configuration file.
    ```bash
    vagrant ssh client
    ```
   Create an `/vagrant/vagrant-client.ovpn` on this instance and paste the contents from the previous step.
1. Open a OpenVPN tunnel on the client instance.
    ```bash
    sudo openvpn -f /vagrant/vagrant-client.ovpn &
    ```
1. Attempt to ping the target instance from the client instance again.
    ```bash
    ping 10.88.111.11
    ```

And voila you should be connected!
Perform additional tests such as checking the Prometheus metrics, etc.
