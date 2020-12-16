package doorman

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	pb "github.com/equinix/doorman/protobuf"
)

var openvpnStatusFile = `TITLE,OpenVPN 2.3.11 x86_64-redhat-linux-gnu [SSL (OpenSSL)] [LZO] [EPOLL] [PKCS11] [MH] [IPv6] built on May 10 2016
TIME,Wed Jun 29 11:52:04 2016,1467215524
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username
CLIENT_LIST,client1,24.255.233.90:60710,192.168.127.2,4497,4990,Wed Jun 29 11:50:04 2016,1467215404,nathan.goulding@gmail.com
CLIENT_LIST,client2,90.255.233.24:70610,192.168.127.3,9744,9049,Thu Jun 30 01:50:04 2015,1367215404,another.user@gmail.com
HEADER,ROUTING_TABLE,Virtual Address,Common Name,Real Address,Last Ref,Last Ref (time_t)
ROUTING_TABLE,192.168.127.2,client,24.255.233.90:60710,Wed Jun 29 11:50:32 2016,1467215432
GLOBAL_STATS,Max bcast/mcast queue length,0
END`

var openSSLIndexFile = "V\t260713172635Z\t\t01\tunknown\t/CN=server\n" +
	"R\t260713193411Z\t160726174738Z\t02\tunknown\t/CN=nathangoulding\n" +
	"R\t260718140125Z\t160720140203Z\t03\tunknown\t/CN=f639bdef-2014-4b6d-ba23-18ee3d91631d\n" +
	"R\t260718140347Z\t160726174430Z\t04\tunknown\t/CN=f639bdef-2014-4b6d-ba23-18ee3d91631d\n" +
	"V\t150724175012Z\t\t05\tunknown\t/CN=nathangoulding\n" +
	"V\t260724210614Z\t\t06\tunknown\t/CN=blah"

func TestExtractCertificate(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "doorman_crt_test")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(f.Name())
	ioutil.WriteFile(f.Name(), []byte("bla\nblah\n-----BEGIN CERTIFICATE-----\nafter"), 0644)

	cert := ExtractCertificate(f.Name())
	if cert != "-----BEGIN CERTIFICATE-----\nafter" {
		t.Fatalf("Cert not equal to what we expect:\n%s", cert)
	}
}

func TestParseOpenSSLIndexFile(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "doorman_index_test")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(f.Name())

	ioutil.WriteFile(f.Name(), []byte(openSSLIndexFile), 0644)

	clients := ClientsFromOpenSSLIndexFile(f.Name())

	if len(clients) != 5 {
		t.Fatalf("expecting clients to be 5, got: %d", len(clients))
	}

	for idx, client := range clients {
		switch idx {
		case 4:
			if client.Status != pb.ClientStatus_VALID {
				t.Fatal("expecting valid")
			}
			break
		case 0:
		case 1:
		case 2:
			if client.Status != pb.ClientStatus_REVOKED {
				t.Fatal("expecting revoked")
			}
			break
		case 3:
			if client.Status != pb.ClientStatus_EXPIRED {
				t.Fatal("expecting expired")
			}
			break
		}
	}
}

func TestIsTestingEnvironment(t *testing.T) {
	// Store environment variable value
	existingEnvironmentValue := os.Getenv(doormanEnvironment)

	// Test Structure
	type test struct {
		environmentVariable string
		want                bool
	}

	// List of tests and the test values.
	tests := []test{
		{environmentVariable: "", want: false},
		{environmentVariable: "production", want: false},
		{environmentVariable: "testing", want: true},
	}

	// Running the tests.
	for _, tc := range tests {
		os.Setenv(doormanEnvironment, tc.environmentVariable)
		result := isTestingEnvironment()
		if result != tc.want {
			t.Fatalf("expected: %v, got: %v", tc.want, result)
		}
	}

	// Clean up, set the environment variable back to its previous value.
	os.Setenv(doormanEnvironment, existingEnvironmentValue)
}
