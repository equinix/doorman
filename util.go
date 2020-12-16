package doorman

import (
	"bufio"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	pb "github.com/equinix/doorman/protobuf"
	"github.com/pkg/errors"
)

var (
	extractCertRe = regexp.MustCompile(`(?ms)^(.*)(-----BEGIN CERTIFICATE-----)(.*)$`)
	twofactorRe   = regexp.MustCompile(`^[0-9]{6}$`)
)

func ParseOpenVPNFile(filename string) (string, string, string, error) {
	var username string
	var password string
	var twofactor string

	if filename == "" {
		return "", "", "", errors.New("no filename supplied")
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", "", "", errors.New("file does not exist")
	}

	inFile, err := os.Open(filename)
	if err != nil {
		return "", "", "", errors.Wrap(err, "open openvpn file")
	}
	defer inFile.Close()

	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		if username == "" {
			username = scanner.Text()
		} else if password == "" {
			password = scanner.Text()
		} else {
			return "", "", "", errors.New("invalid formatted file, expecting only 2 lines")
		}
	}

	if username == "" {
		return "", "", "", errors.New("empty username")
	}
	if password == "" || len(password) <= 6 {
		return "", "", "", errors.New("invalid password")
	}

	twofactor = password[0:6]
	password = password[6:]

	if matched := twofactorRe.Match([]byte(twofactor)); !matched {
		return "", "", "", errors.Errorf("invalid twofactor token: %s", twofactor)
	}

	return username, password, twofactor, nil
}

func ExtractCertificate(file string) string {
	// logs in here because caller doesn't check return
	cert, err := ioutil.ReadFile(file)
	if err != nil {
		logger.With("file", file).Error(errors.Wrap(err, "read certificate"))
		return ""
	}

	return strings.TrimSpace(extractCertRe.ReplaceAllString(string(cert), "$2$3"))
}

func ExtractOpenVPNCA(file string) string {
	// logs in here because caller doesn't check return
	ca, err := ioutil.ReadFile(file)
	if err != nil {
		logger.With("file", file).Error(errors.Wrap(err, "read ca file"))
		return ""
	}

	return strings.TrimSpace(string(ca))
}

func ExtractPrivateKey(file string) string {
	// logs in here because caller doesn't check return
	key, err := ioutil.ReadFile(file)
	if err != nil {
		logger.With("file", file).Error(errors.Wrap(err, "read key file"))
		return ""
	}

	return strings.TrimSpace(string(key))
}

func ClientFromOpenSSLIndexFile(client string) *pb.Client {
	for _, c := range ClientsFromOpenSSLIndexFile("") {
		if c.Client == client {
			return c
		}
	}
	return &pb.Client{}
}

func ClientsFromOpenSSLIndexFile(file string) []*pb.Client {
	if file == "" {
		file = doormanEasyRSADir + "/pki/index.txt"
	}

	indexFile, err := ioutil.ReadFile(file)
	if err != nil {
		logger.With("file", file).Error(errors.Wrap(err, "read OpenSSL index file"))
		return nil
	}

	clients := []*pb.Client{}

	// 26:07:24 17:50:12 Z
	//            260724175012Z
	dateLayout := "060102150405Z"

	for _, line := range strings.Split(strings.TrimSpace(string(indexFile)), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 6 || fields[5] == "/CN=server" {
			continue
		}
		expiresDate := int64(0)
		revocationDate := int64(0)
		status := pb.ClientStatus_EXPIRED
		if fields[0] == "R" {
			status = pb.ClientStatus_REVOKED
		} else {
			date, _ := time.Parse(dateLayout, fields[1])
			expiresDate = date.Unix()
			if expiresDate > time.Now().Unix() {
				status = pb.ClientStatus_VALID
			}
		}
		if len(fields[2]) > 0 {
			date, _ := time.Parse(dateLayout, fields[2])
			revocationDate = date.Unix()
		}
		client := &pb.Client{
			Status:         status,
			ExpiresDate:    expiresDate,
			RevocationDate: revocationDate,
			Client:         strings.Replace(fields[5], "/CN=", "", 1),
		}
		clients = append(clients, client)
	}

	return clients
}

// This can likely be modified at a later time to just check to see is an environment variable has any value.
func isTestingEnvironment() bool {
	return os.Getenv(doormanEnvironment) == "testing"
}
