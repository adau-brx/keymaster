/*
Package aws_role may be used by service code to obtain Keymaster-issued identity
certificates. The identity certificate will contain the AWS IAM role that the
service code is able to assume (i.e. EC2 instance profile, EKS IRSA, Lambda
role). The full AWS Role ARN is stored in a certificate URI SAN extension and a
simplified form of the ARN is stored in the certificate CN.

The service code does not require any extra permissions. It uses the
sts:GetCallerIdentity permission that is available to all AWS identities. Thus,
no policy configuration is required.

This code uses the AWS IAM credentials to request a pre-signed URL from the AWS
Security Token Service (STS). This pre-signed URL is passed to Keymaster which
can make a request using the URL to verify the identity of the caller. No
credentials are sent.
*/
package aws_role

import (
	"context"
	"crypto"
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Params struct {
	// Required parameters.
	KeymasterServer string
	Logger          log.DebugLogger
	// Optional parameters.
	Context          context.Context
	HttpClient       *http.Client
	Signer           crypto.Signer
	awsConfig        aws.Config
	derPubKey        []byte
	isSetup          bool
	pemPubKey        []byte
	roleArn          string
	stsClient        *sts.Client
	stsPresignClient *sts.PresignClient
}

type Manager struct {
	params    Params
	mutex     sync.RWMutex // Protect everything below.
	certError error
	certPEM   []byte
	certTLS   *tls.Certificate
	waiters   map[chan<- struct{}]struct{}
}

// GetRoleCertificate requests an AWS role identify certificate from the
// Keymaster server specified in params. It returns the certificate PEM.
func GetRoleCertificate(params Params) ([]byte, error) {
	return params.getRoleCertificate()
}

// GetRoleCertificateTLS requests an AWS role identify certificate from the
// Keymaster server specified in params. It returns the certificate.
func GetRoleCertificateTLS(params Params) (*tls.Certificate, error) {
	_, certTLS, err := params.getRoleCertificateTLS()
	return certTLS, err
}

// NewManager returns a certificate manager which provides AWS role identity
// certificates from the Keymaster server specified in params. Certificates
// are refreshed in the background.
func NewManager(params Params) (*Manager, error) {
	return newManager(params)
}

// GetClientCertificate returns a valid, cached certificate. The method
// value may be assigned to the crypto/tls.Config.GetClientCertificate field.
func (m *Manager) GetClientCertificate(cri *tls.CertificateRequestInfo) (
	*tls.Certificate, error) {
	return m.getClientCertificate(cri)
}

// GetRoleCertificate returns a valid, cached certificate. It returns the
// certificate PEM, TLS certificate and error.
func (m *Manager) GetRoleCertificate() ([]byte, *tls.Certificate, error) {
	return m.getRoleCertificate()
}

// WaitForRefresh waits until a successful certificate refresh.
func (m *Manager) WaitForRefresh() {
	m.waitForRefresh()
}
