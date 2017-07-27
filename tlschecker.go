package checkup

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"
)

// TLSChecker implements a Checker for TLS endpoints.
type TLSChecker struct {
	// Name is the name of the endpoint.
	Name string `json:"endpoint_name"`

	// DomainName is the domain name of the endpoint.
	DomainName string `json:"domain_name"`

	// Port is the port of the endpoint.
	// If omitted, the default value is 443
	Port int `json:"port,omitempty"`

	// Threshold is the number of days
	// before the certificate expiration date
	// required for notification to be fired
	Threshold int `json:"threshold,omitempty"`

	// Attempts is how many requests the client will
	// make to the endpoint in a single check.
	Attempts int `json:"attempts,omitempty"`

	// Tags are custom tags providing context to the check
	Tags map[string]string `json:"tags,omitempty"`
}

// CertProperties is SSL certificate properties
type CertProperties struct {
	CommonName string
	Serial     *big.Int
	NotBefore  time.Time
	NotAfter   time.Time
	DNSNames   []string
	Issuer     string
}

// Check performs checks using c according to its configuration.
// An error is only returned if there is a configuration error.
func (c TLSChecker) Check() (Result, error) {
	var p *CertProperties
	if c.Attempts < 1 {
		c.Attempts = 1
	}

	result := Result{Title: c.Name, Endpoint: c.DomainName, Timestamp: Timestamp()}
	if c.Port == 0 {
		c.Port = 443
	}
	certURL := c.DomainName + ":" + fmt.Sprint(c.Port)

	p, result.Times = c.doChecks(certURL)
	result.Type = "tls"
	if p != nil {
		result = c.conclude(p, result)
	} else {
		result.Down = true
		result.Notice = fmt.Sprint(result.Times)
		p = &CertProperties{
			CommonName: c.DomainName,
			Serial:     big.NewInt(0),
			NotBefore:  time.Time{},
			NotAfter:   time.Time{},
			DNSNames:   []string{"N/A"},
			Issuer:     "N/A",
		}
	}
	result.Context = *p
	if c.Tags != nil {
		result.Tags = c.Tags
	}

	return result, nil
}

// doChecks executes certificate check and returns each attempt
func (c TLSChecker) doChecks(certURL string) (*CertProperties, Attempts) {
	var p *CertProperties
	checks := make(Attempts, c.Attempts)
	for i := 0; i < c.Attempts; i++ {
		start := time.Now()
		certs, err := getCert(&certURL)
		checks[i].RTT = time.Since(start)
		if err != nil {
			checks[i].Error = err.Error()
			continue
		}
		p = parseCert(certs)
	}
	return p, checks
}

// conclude takes NotAfter in certificate property
// and checks it against the checker's Threshold.
// If the certificate is expired its status is down,
// if less then the threshold - degraded
func (c TLSChecker) conclude(p *CertProperties, result Result) Result {

	daysToExpire := -1 * int(time.Since(p.NotAfter).Hours()/24)

	// Healthy
	if daysToExpire > c.Threshold {
		result.Healthy = true
		return result
	}
	// Expiring (degraded)
	if daysToExpire > 0 {
		result.Degraded = true
		return result
	}
	// Expired (down)
	result.Down = true
	return result
}

func getCert(certURL *string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	config := tls.Config{InsecureSkipVerify: true}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(&dialer, "tcp", *certURL, &config)
	if err != nil {
		log.Println("Failed to connect: " + err.Error())
		return certs, err
	}
	defer conn.Close()

	state := conn.ConnectionState()
	certs = state.PeerCertificates

	return certs, nil
}

func parseCert(certs []*x509.Certificate) *CertProperties {
	var p CertProperties
	for _, v := range certs {
		cert, err := x509.ParseCertificate(v.Raw)
		if err != nil {
			log.Fatalf("failed to parse certificate: " + err.Error())
		}
		if cert.DNSNames == nil {
			continue
		}
		p = CertProperties{
			CommonName: cert.Subject.CommonName,
			Serial:     cert.SerialNumber,
			NotBefore:  cert.NotBefore,
			NotAfter:   cert.NotAfter,
			DNSNames:   cert.DNSNames,
			Issuer:     cert.Issuer.CommonName,
		}
		return &p
	}
	return nil
}
