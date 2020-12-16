package doormanc

import (
	"crypto/x509"
	"os"

	doorman "github.com/equinix/doorman/protobuf"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func New(facility string) (doorman.VPNServiceClient, error) {
	auth := grpc.WithTransportCredentials(credentials.NewTLS(nil))

	cert := os.Getenv("GRPC_CERT")
	insecure := os.Getenv("GRPC_INSECURE")
	if cert != "" && insecure != "" {
		return nil, errors.New("GRPC_CERT and GRPC_INSECURE are mutually exclusive but both are set")
	}

	if cert != "" {
		cp := x509.NewCertPool()
		ok := cp.AppendCertsFromPEM([]byte(cert))
		if !ok {
			return nil, errors.New("unable to parse cert")
		}
		auth = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(cp, ""))
	}
	if os.Getenv("GRPC_INSECURE") != "" {
		auth = grpc.WithInsecure()
	}

	host := "127.0.0.1"
	if facility != "" {
		host = "doorman-" + facility + ".equinix.com"
	}

	conn, err := grpc.Dial(host+":8080", auth)
	if err != nil {
		return nil, errors.Wrap(err, "dial server")
	}

	// conn should be closed...
	return doorman.NewVPNServiceClient(conn), nil
}
