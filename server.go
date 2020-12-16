package doorman

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	retryable "github.com/hashicorp/go-retryablehttp"
	"github.com/equinix/doorman/metrics"
	pb "github.com/equinix/doorman/protobuf"
	"github.com/packethost/packngo"
	"github.com/packethost/pkg/grpc"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// environment variables
	doormanConsumerToken = "DOORMAN_CONSUMER_TOKEN"
	doormanApiHost       = "DOORMAN_API_HOST"
	doormanEnvironment   = "EQUINIX_ENV"
	doormanFacilityCode  = "FACILITY"
	doormanMagicIP       = "DOORMAN_MAGIC_IP"
	promethuesServerPort = "PROMETHUES_SERVER_PORT"

	doormanOpenVPNCCD = "/etc/openvpn/ccd" // client-config-directory
	doormanEasyRSADir = "/etc/openvpn/easy-rsa"
	easyrsa           = "/usr/share/easy-rsa/easyrsa"
)

const (
	commandCreate = "/app/fw-add.sh create %s"
	commandAdd    = "/app/fw-add.sh add %s %s/%d"
	commandEnable = "/app/fw-add.sh enable %s %s"

	commandDisable = "/app/fw-del.sh disable %s %s"
	commandDelete  = "/app/fw-del.sh delete %s"
)

type AuthToken struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type VPNServer struct {
	magicIP       string
	facilityCode  string
	consumerToken string
	apiHost       string
	sessions      *url.URL

	mu          sync.RWMutex
	allocations []pb.Allocation
	connections map[string]*pb.Connection
}

// MARK: implement VPNService (vpn_service.pb.go)
func (s *VPNServer) ListAllocations(ctx context.Context, in *pb.ListAllocationsRequest) (*pb.ListAllocationsResponse, error) {
	logger.Info("got list allocations request")
	s.mu.RLock()
	defer s.mu.RUnlock()

	allocations := make([]*pb.Allocation, 0, len(s.allocations))
	for _, allocation := range s.allocations {
		if !in.OnlyAllocated || allocation.Client != "" {
			allocations = append(allocations, &pb.Allocation{Client: allocation.Client, Ip: allocation.Ip})
		}
	}

	response := &pb.ListAllocationsResponse{
		Allocations: allocations,
	}
	return response, nil
}

// scrubURLError will replace a password query-param's value with "******" (8 actual * characters) only if the error is of type *url.Error
// non *url.Errors or url.Errors w/o "password" are returned with no change.
func scrubURLError(u url.URL, err error) error {
	const pass = "********"
	if e, ok := err.(*url.Error); !ok {
		return err
	} else {
		q := u.Query()
		if q.Get("password") == "" {
			return err
		}
		q.Set("password", pass)
		u.RawQuery = q.Encode()
		e.URL = u.String()
		return e
	}
}

func fetchProjects(client *packngo.Client) ([]packngo.Project, error) {
	projects, _, err := client.Projects.List(nil)
	if err != nil {
		return nil, errors.Wrap(err, "listing projects")
	}
	if len(projects) == 0 {
		return nil, errors.New("no projects found")
	}
	return projects, nil
}

func fetchIPs(client *packngo.Client, facility string, project packngo.Project) ([]packngo.IPAddressReservation, error) {
	pIPs, _, err := client.ProjectIPs.List(project.ID, &packngo.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "fetching ip addresses of project=%s", project.ID)
	}

	var ips []packngo.IPAddressReservation
	for _, ip := range pIPs {
		if ip.Public {
			continue
		}
		ips = append(ips, ip)
	}

	return ips, nil
}

func getSubnets(client *packngo.Client, facility string) ([]packngo.IPAddressReservation, error) {

	var ips []packngo.IPAddressReservation

	if isTestingEnvironment() {
		ip := packngo.IpAddressCommon{Address: "10.88.111.11", Gateway: "10.88.111.1", Network: "10.88.111.0", AddressFamily: 4, Netmask: "255.255.255.128", Public: false, CIDR: 25, Management: false, Manageable: true}
		ipres := packngo.IPAddressReservation{IpAddressCommon: ip}
		ips = append(ips, ipres)
		return ips, nil
	}

	projects, err := fetchProjects(client)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(len(projects))
	errs := make(chan error)

	var mu sync.Mutex

	for _, project := range projects {
		go func(project packngo.Project) {
			defer wg.Done()

			pIPs, err := fetchIPs(client, facility, project)
			if err != nil {
				errs <- err
			}
			if len(pIPs) == 0 {
				return
			}

			mu.Lock()
			ips = append(ips, pIPs...)
			mu.Unlock()

		}(project)
	}

	wg.Wait()
	close(errs)
	err = <-errs
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, errors.New("no backend routes to push")
	}

	return ips, nil
}

func (s *VPNServer) configureClient(log log.Logger, w io.StringWriter, client string, ips []packngo.IPAddressReservation) (alloc *pb.Allocation, routes []*pb.Route, err error) {
	allocation, err := s.reserveNextAvailableIP(client)
	if err != nil {
		err = errors.WithMessage(err, "reserve next available ip")
		log.Error(err)
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			return
		}

		s.freeIPAllocation(client)
		allocation = nil
	}()

	vpnIP := allocation.Ip

	cmd := fmt.Sprintf(commandCreate, vpnIP)
	if stdout, stderr, err := s.shellRun(cmd); err != nil {
		log.With("stdout", stdout, "stderr", stderr).Error(err)
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			return
		}

		cmd := fmt.Sprintf(commandDelete, vpnIP)
		if stdout, stderr, cleanupErr := s.shellRun(cmd); cleanupErr != nil {
			log.With("stdout", stdout, "stderr", stderr, "cmd", cmd).Fatal(cleanupErr, "error while cleaning up an error")
			metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		}

		routes = nil
	}()

	for _, ip := range ips {
		cmd = fmt.Sprintf(commandAdd, vpnIP, ip.Network, ip.CIDR)
		if stdout, stderr, err := s.shellRun(cmd); err != nil {
			log.With("stdout", stdout, "stderr", stderr).Error(err)
			// clean up is already handled in commandCreate's defer
			metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
			return nil, nil, err
		}
		log.Debug(cmd)

		routes = append(routes, &pb.Route{Cidr: fmt.Sprintf("%s/%d", ip.Network, ip.CIDR)})

		route := fmt.Sprintf(`push "route %s %s"`+"\n", ip.Network, ip.Netmask)
		log.Debug(route)
		w.WriteString(route)
	}

	cmd = fmt.Sprintf(commandEnable, vpnIP, s.magicIP)
	if stdout, stderr, err := s.shellRun(cmd); err != nil {
		log.With("stdout", stdout, "stderr", stderr).Error(err)
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, nil, err
	}

	w.WriteString(fmt.Sprintln("ifconfig-push", vpnIP, "255.255.255.0"))

	return allocation, routes, nil
}

// Private helper functions for the Authenticate method
func (s *VPNServer) createEncodedURL(username string, password string) url.URL {
	loginURL := *s.sessions
	loginURL.RawQuery = url.Values(map[string][]string{
		"login":    {username},
		"password": {password},
	}).Encode()

	return loginURL
}

func (s *VPNServer) createLoginRequest(loginURL url.URL) (*retryable.Request, error) {
	req, err := retryable.NewRequest("POST", loginURL.String(), nil)
	if err != nil {
		err = errors.Wrap(err, "failed setting up http request")
		logger.With("error", err).Info()
		return nil, err
	}
	return req, nil
}

func (s *VPNServer) validate2faEnabled(loginURL url.URL, request *retryable.Request) error {
	// Set Token
	request.Header.Set("X-Consumer-Token", s.consumerToken)

	client := retryable.NewClient()

	logger.Info("attempting to validate user token.")
	response, err := client.Do(request)
	if err != nil {
		err = errors.Wrap(scrubURLError(loginURL, err), "failed to connect to service.")
		logger.With("error", err).Info()
		return err
	}

	// User name and password are valid, but no 2fa has been enabled.
	// Login was successful.
	if response.StatusCode/100 == 2 {
		err = errors.New("2-factor not enabled")
		logger.With("error", err).Info()
		metrics.AuthenticationFailureTotalCount.Inc()
		return err
	}

	return nil
}

func (s *VPNServer) validateCredentials(loginURL url.URL, request *retryable.Request, twoFactorToken string) (*http.Response, error) {
	// Set OTP Token
	request.Header.Set("X-OTP-Token", twoFactorToken)

	client := retryable.NewClient()

	logger.Info("attempting to validate user token.")
	response, err := client.Do(request)
	if err != nil {
		err = errors.Wrap(scrubURLError(loginURL, err), "failed to validate customer token")
		logger.With("error", err).Info()
		return nil, err
	}

	if response.StatusCode/100 != 2 {
		err := errors.New("invalid username, password, or 2-factor token`")
		logger.With("error", err, "response", response.StatusCode).Info()
		metrics.AuthenticationFailureTotalCount.Inc()
		return nil, err
	}
	return response, nil
}

func (s *VPNServer) Authenticate(ctx context.Context, in *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	if in.Client == "" {
		return nil, errors.New("no OpenVPN client supplied")
	}

	log := logger.With("client", in.Client, "address", in.ConnectingIp)
	log.Info("received authenticate request")

	var authToken *AuthToken = &AuthToken{}
	var username string

	if !isTestingEnvironment() {
		username, password, twofactor, err := ParseOpenVPNFile(in.File)
		if err != nil {
			err = errors.WithMessage(err, "parse openvpn file")
			log.With("error", err).Info()
			return nil, err
		}

		encodedURL := s.createEncodedURL(username, password)
		request, err := s.createLoginRequest(encodedURL)
		if err != nil {
			return nil, err
		}

		start := time.Now()
		err = s.validate2faEnabled(encodedURL, request)
		if err != nil {
			return nil, err
		}
		response, err := s.validateCredentials(encodedURL, request, twofactor)
		if err != nil {
			return nil, err
		}
		err = json.NewDecoder(response.Body).Decode(authToken)
		if err != nil || authToken.Token == "" {
			err = errors.Wrap(err, "fetching API authentication token")
			logger.With("error", err).Info()
			return nil, err
		}
		if err != nil {
			return nil, err
		}

		duration := time.Since(start)
		metrics.AuthenticationDuration.Observe(duration.Seconds())
		metrics.AuthenticationSuccessTotalCount.Inc()
		log.Info("successfully authenticated client")
	} else {
		username = "00000000-0000-0000-0000-000000000001"
	}

	s.mu.RLock()
	_, ok := s.connections[in.Client]
	s.mu.RUnlock()
	if ok {
		log.Info("client seems to already be connected, maybe we were slow last time around")
		return &pb.AuthenticateResponse{Status: 0}, nil
	}

	packetClient := packngo.NewClientWithAuth(s.consumerToken, authToken.Token, nil)

	ips, err := getSubnets(packetClient, s.facilityCode)
	if err != nil {
		log.With("err", err).Info()
		return nil, err
	}

	ccdFile, err := os.Create(doormanOpenVPNCCD + "/" + in.Client)
	if err != nil {
		err = errors.Wrap(err, "creating openvpn config file")
		log.Error(err)
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, err
	}
	defer ccdFile.Close()

	allocation, routes, err := s.configureClient(log, ccdFile, in.Client, ips)
	if err != nil {
		log.With("error", err).Info("failed to configure client")
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, err
	}

	connection := &pb.Connection{
		Client:       in.Client,
		Allocation:   allocation,
		Routes:       routes,
		Since:        time.Now().Unix(),
		ConnectingIp: in.ConnectingIp,
		Username:     username,
	}
	s.mu.Lock()
	s.connections[in.Client] = connection
	s.mu.Unlock()
	metrics.ActiveClientTotal.Inc()

	return &pb.AuthenticateResponse{Status: 0}, nil
}

func (s *VPNServer) ListConnections(ctx context.Context, in *pb.ListConnectionsRequest) (*pb.ListConnectionsResponse, error) {
	logger.Info("incoming list connections request")

	s.mu.RLock()
	connections := make([]*pb.Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		connections = append(connections, conn)

	}
	s.mu.RUnlock()

	response := &pb.ListConnectionsResponse{
		Total:       int32(len(connections)),
		Connections: connections,
	}
	return response, nil
}

func (s *VPNServer) CreateClient(ctx context.Context, in *pb.CreateClientRequest) (*pb.CreateClientResponse, error) {
	logger.With("client", in.Client).Info("got create client request")
	if in.Client == "server" {
		err := errors.New("cannot create `server` certificate")
		logger.With("error", err).Info()
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, err
	}
	if in.Force {
		s.revokeCertificate(in.Client, true)
	}

	cmd := "/app/client-create.sh --client=" + in.Client
	if stdout, stderr, err := s.shellRun(cmd); err != nil {
		err = errors.WithMessage(err, "build-client-full")
		logger.With("stdout", stdout, "stderr", stderr, "error", err).Info()
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, err
	}
	response := &pb.CreateClientResponse{
		Config: s.generateConfig(in.Client),
	}
	return response, nil
}

func (s *VPNServer) GetClient(ctx context.Context, in *pb.GetClientRequest) (*pb.GetClientResponse, error) {
	logger.Info("got get clients request")
	client := ClientFromOpenSSLIndexFile(in.Client)
	if client.Client == "" {
		err := errors.New("invalid client `" + in.Client + "` specified")
		logger.With("error", err).Info()
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, err
	}
	response := &pb.GetClientResponse{
		Status:         client.Status,
		ExpiresDate:    client.ExpiresDate,
		RevocationDate: client.RevocationDate,
		Config:         s.generateConfig(in.Client),
	}
	return response, nil
}

func (s *VPNServer) RevokeClient(ctx context.Context, in *pb.RevokeClientRequest) (*pb.RevokeClientResponse, error) {
	logger.With("client", in.Client).Info("got revoke client request")
	if in.Client == "server" {
		err := errors.New("cannot revoke `server` certificate")
		logger.With("error", err).Info()
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return nil, err

	}
	if err := s.revokeCertificate(in.Client, false); err != nil {
		// error is logged in revokeCertificate
		return nil, err
	}

	response := &pb.RevokeClientResponse{
		Status: 0,
	}

	// If not connected do what?
	if len(s.connections) == 0 {
		logger.Info("no active connections, done with revocation")
		return response, nil
	}
	logger.Info("active connections found, closing them")
	s.disconnect(in.Client)

	return response, nil
}

func (s *VPNServer) ListClients(ctx context.Context, in *pb.ListClientsRequest) (*pb.ListClientsResponse, error) {
	logger.Info("got list clients request")
	response := &pb.ListClientsResponse{
		Clients: ClientsFromOpenSSLIndexFile(""),
	}
	return response, nil
}

func (s *VPNServer) Disconnect(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectResponse, error) {
	logger.With("client", in.Client).Info("got disconnect client request")

	s.disconnect(in.Client)

	response := &pb.DisconnectResponse{
		Status: 0,
	}
	metrics.ActiveClientTotal.Dec()
	return response, nil
}

func (s *VPNServer) disconnect(client string) error {
	if err := s.removeIptables(client); err != nil {
		return err
	}
	s.freeIPAllocation(client)

	s.mu.Lock()
	delete(s.connections, client)
	defer s.mu.Unlock()

	return nil
}

func (s *VPNServer) removeIptables(client string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	connection := s.connections[client]
	vpnIP := connection.Allocation.Ip
	cmd := fmt.Sprintf(commandDisable, vpnIP, s.magicIP)
	if stdout, stderr, err := s.shellRun(cmd); err != nil {
		logger.With("stdout", stdout, "stderr", stderr).Error(err)
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return err
	}

	cmd = fmt.Sprintf(commandDelete, vpnIP)
	if stdout, stderr, err := s.shellRun(cmd); err != nil {
		logger.With("stdout", stdout, "stderr", stderr).Error(err)
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		return err
	}
	return nil
}

func (s *VPNServer) reserveNextAvailableIP(client string) (*pb.Allocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var allocated *pb.Allocation
	for i := range s.allocations {
		alloc := &s.allocations[i]
		// we only allow a single connection per client, so if they are already connected
		// return the same IP address
		if alloc.Client == client {
			allocated = alloc
			break
		}
		// return the next available IP address and assign it to them
		if alloc.Client == "" {
			alloc.Client = client
			allocated = alloc
			break
		}
	}
	if allocated != nil {
		return allocated, nil
	}
	return nil, errors.New("no available ips in pool")
}

func (s *VPNServer) freeIPAllocation(client string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.allocations {
		allocation := &s.allocations[i]
		if allocation.Client == client {
			logger.With("ip", allocation.Ip, "client", allocation.Client).Info("freed IP allocation")
			allocation.Client = ""
			break
		}
	}
}

func (s *VPNServer) generateConfig(client string) string {
	const config = `client
server-poll-timeout 4
nobind
remote vpn.%s.platformequinix.com 1194 tcp
dev tun
dev-type tun
reneg-sec 604800
sndbuf 100000
rcvbuf 100000
auth-user-pass
comp-lzo no
verb 3
setenv PUSH_PEER_INFO

<ca>
%s
</ca>

<cert>
%s
</cert>

<key>
%s
</key>`

	return fmt.Sprintf(config,
		s.facilityCode,
		ExtractOpenVPNCA(doormanEasyRSADir+"/pki/ca.crt"),
		ExtractCertificate(doormanEasyRSADir+"/pki/issued/"+client+".crt"),
		ExtractPrivateKey(doormanEasyRSADir+"/pki/private/"+client+".key"),
	)
}

func (s *VPNServer) revokeCertificate(client string, ignoreError bool) error {
	cmd := "/app/client-revoke.sh "
	if ignoreError {
		cmd += "--ignore-errors "
	}
	cmd += "--client=" + client

	logger.Info("running client-revoke on client=" + client)
	stdout, stderr, err := s.shellRun(cmd)
	if err != nil && !ignoreError {
		if ignoreError {
			err = nil
		} else {
			logger.With("stdout", stdout, "stderr", stderr).Error(errors.WithMessage(err, "run revoke"))
		}
	}
	return err
}

func (s *VPNServer) shellRunBackground(ctx context.Context, l log.Logger, cmd string) error {
	c := exec.CommandContext(ctx, "/bin/bash", "-c", cmd)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "setup stdout pipe")
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "setup stderr pipe")
	}

	err = c.Start()
	if err != nil {
		return errors.Wrap(err, "start command")
	}

	var wg sync.WaitGroup
	wg.Add(2)

	scanner := func(s *bufio.Scanner, handle string) {
		for s.Scan() {
			l.With(handle, true).Info(s.Text())
		}
		if err := s.Err(); err != nil {
			l.Error(errors.Wrap(s.Err(), "handling "+handle))
		}
		wg.Done()
	}

	scanOut := bufio.NewScanner(stdout)
	go func() {
		scanner(scanOut, "stdout")
	}()

	scanErr := bufio.NewScanner(stderr)
	go func() {
		scanner(scanErr, "stderr")
	}()

	err = c.Wait()
	if err != nil && ctx.Err() != context.Canceled {
		return errors.Wrap(err, "run command")
	}
	return nil
}

func (s *VPNServer) shellRun(cmd string) (string, string, error) {
	c := exec.Command("/bin/bash", "-c", cmd)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		err = errors.Wrap(err, "run command")
	}
	return stdout.String(), stderr.String(), err
}

func (s *VPNServer) startOpenVPN(ctx context.Context) chan error {
	logger.Info("starting openvpn")
	ch := make(chan error)
	go func() {
		if err := s.shellRunBackground(ctx, logger, "/usr/sbin/openvpn --config /etc/openvpn/server.conf"); err != nil {
			metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
			logger.Fatal(errors.WithMessage(err, "start openvpn"))
		}
		logger.Info("openvpn is done")
		close(ch)
	}()
	return ch
}

func (s *VPNServer) setupFirewall() {
	if stdout, stderr, err := s.shellRun("/app/fw-init.sh"); err != nil {
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		logger.With("stdout", stdout, "stderrr", stderr).Fatal(errors.Wrap(err, "initialize firewall setup"))
	}
}

// This populates our IPAllocations to fetch from when clients connect
func (s *VPNServer) populateIPAllocationPool() {
	s.mu.Lock()
	defer s.mu.Unlock()

	ip, network, _ := net.ParseCIDR("192.168.127.1/24")
	var rawIPs []string
	for rawIP := ip.Mask(network.Mask); network.Contains(rawIP); s.nextIP(rawIP) {
		rawIPs = append(rawIPs, rawIP.String())
	}

	// remove 192.168.127.0, 192.168.127.1, and 192.168.127.255
	rawIPs = rawIPs[2 : len(rawIPs)-1]
	for _, allocationIP := range rawIPs {
		s.allocations = append(s.allocations, pb.Allocation{Ip: allocationIP})
	}
}

//  http://play.golang.org/p/m8TNTtygK0
func (s *VPNServer) nextIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (s *VPNServer) Serve() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	ctx, cancel := context.WithCancel(context.Background())

	s.populateIPAllocationPool()
	s.setupFirewall()
	ovpn := s.startOpenVPN(ctx)

	req := func(server *grpc.Server) {
		pb.RegisterVPNServiceServer(server.Server(), s)
	}

	var ops []grpc.Option
	if os.Getenv("GRPC_INSECURE") == "" {
		doormanServerCrtFile := "/etc/openvpn/doorman-grpc/server-" + s.facilityCode + ".crt"
		doormanServerKeyFile := "/etc/openvpn/doorman-grpc/server-" + s.facilityCode + ".key"
		ops = append(ops, grpc.LoadX509KeyPair(doormanServerCrtFile, doormanServerKeyFile))
	}

	server, err := grpc.NewServer(logger, req, ops...)
	if err != nil {
		metrics.ErrorTotal.WithLabelValues("doorman", "errors").Inc()
		logger.Fatal(errors.Wrap(err, "start grpc server"))
	}

	go func() {
		sig := <-sigs
		logger.With("signal", sig).Info("exiting due to signal")
		cancel()
		server.Server().GracefulStop()
	}()
	go func() {
		<-ovpn
		cancel()
		server.Server().GracefulStop()
	}()

	logger.Info("serving grpc")
	server.Serve()
	<-ovpn
}

func ServeVPN(l log.Logger) {
	logger = l.Package("serve")

	metrics.Init()
	http.Handle("/metrics", promhttp.Handler())

	prometheusPort := os.Getenv(promethuesServerPort)
	if prometheusPort == "" {
		logger.Info("The environment variable 'PROMETHUES_SERVER_PORT' was not set, defaulting to 9090")
		prometheusPort = ":9090"
	}

	apiHost := os.Getenv(doormanApiHost)
	if apiHost == "" {
		logger.Fatal(errors.New(doormanApiHost + " is empty"))
	}

	sessions, err := url.Parse(apiHost + "/sessions")
	if err != nil {
		logger.Fatal(errors.Wrap(err, "parsing api sessions url"))
	}

	consumerToken := os.Getenv(doormanConsumerToken)
	if consumerToken == "" {
		logger.Fatal(errors.New(doormanConsumerToken + " is empty"))
	}

	magicIP := os.Getenv(doormanMagicIP)

	facilityCode := os.Getenv(doormanFacilityCode)
	if facilityCode == "" {
		logger.Fatal(errors.New(doormanFacilityCode + " is empty"))
	}

	server := &VPNServer{
		magicIP:       magicIP,
		facilityCode:  facilityCode,
		apiHost:       apiHost,
		sessions:      sessions,
		consumerToken: consumerToken,
		connections:   map[string]*pb.Connection{},
	}

	go http.ListenAndServe(prometheusPort, nil)
	server.Serve()
}
