package constructors

import (
	"crypto/ecdsa"
	"encoding/pem"
	"io/ioutil"

	"go.uber.org/zap"

	"github.com/headingy/trireme"
	"github.com/headingy/trireme/configurator"
	"github.com/headingy/trireme/crypto"
	"github.com/headingy/trireme/enforcer"
	"github.com/headingy/trireme/enforcer/utils/pkiverifier"
	"github.com/headingy/trireme/monitor"
	"github.com/headingy/trireme/monitor/dockermonitor"

	"github.com/headingy/trireme-example/policyexample"
)

var (
	// ExternalProcessor to use if needed
	ExternalProcessor enforcer.PacketProcessor
)

// TriremeWithPKI is a helper method to created a PKI implementation of Trireme
func TriremeWithPKI(keyFile, certFile, caCertFile string, networks []string, extractor *dockermonitor.DockerMetadataExtractor, remoteEnforcer bool, killContainerError bool) (trireme.Trireme, monitor.Monitor) {

	// Load client cert
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	// Load key
	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		zap.L().Fatal("Failed to read key PEM")
	}

	// Load CA cert
	caCertPEM, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	policyEngine := policyexample.NewCustomPolicyResolver(networks)

	t, m, p := configurator.NewPKITriremeWithDockerMonitor("Server1", policyEngine, ExternalProcessor, nil, false, keyPEM, certPEM, caCertPEM, *extractor, remoteEnforcer, killContainerError)

	if err := p.PublicKeyAdd("Server1", certPEM); err != nil {
		zap.L().Fatal(err.Error())
	}

	return t, m
}

// TriremeWithPSK is a helper method to created a PSK implementation of Trireme
func TriremeWithPSK(networks []string, extractor *dockermonitor.DockerMetadataExtractor, remoteEnforcer bool, killContainerError bool) (trireme.Trireme, monitor.Monitor) {

	policyEngine := policyexample.NewCustomPolicyResolver(networks)

	// Use this if you want a pre-shared key implementation
	return configurator.NewPSKTriremeWithDockerMonitor("Server1", policyEngine, ExternalProcessor, nil, false, []byte("THIS IS A BAD PASSWORD"), *extractor, remoteEnforcer, killContainerError)
}

// HybridTriremeWithPSK is a helper method to created a PSK implementation of Trireme
func HybridTriremeWithPSK(networks []string, extractor *dockermonitor.DockerMetadataExtractor, killContainerError bool) (trireme.Trireme, monitor.Monitor, monitor.Monitor) {

	policyEngine := policyexample.NewCustomPolicyResolver(networks)

	pass := []byte("THIS IS A BAD PASSWORD")
	// Use this if you want a pre-shared key implementation
	return configurator.NewPSKHybridTriremeWithMonitor("Server1", policyEngine, ExternalProcessor, nil, false, pass, *extractor, killContainerError)
}

// HybridTriremeWithCompactPKI is a helper method to created a PKI implementation of Trireme
func HybridTriremeWithCompactPKI(keyFile, certFile, caCertFile, caKeyFile string, networks []string, extractor *dockermonitor.DockerMetadataExtractor, remoteEnforcer bool, killContainerError bool) (trireme.Trireme, monitor.Monitor) {

	// Load client cert
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	// Load key
	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		zap.L().Fatal("Failed to read key PEM")
	}

	// Load CA cert
	caCertPEM, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	caKeyPEM, err := ioutil.ReadFile(caKeyFile)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	token, err := createTxtToken(caKeyPEM, caCertPEM, certPEM)
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	policyEngine := policyexample.NewCustomPolicyResolver(networks)

	return configurator.NewCompactPKIWithDocker("Server1", policyEngine, ExternalProcessor, nil, false, keyPEM, certPEM, caCertPEM, token, *extractor, remoteEnforcer, killContainerError)

}

func createTxtToken(caKeyPEM, caPEM, certPEM []byte) ([]byte, error) {
	caKey, err := crypto.LoadEllipticCurveKey(caKeyPEM)
	if err != nil {
		return nil, err
	}
	caCert, err := crypto.LoadCertificate(caPEM)
	if err != nil {
		return nil, err
	}

	clientCert, err := crypto.LoadCertificate(certPEM)
	if err != nil {
		return nil, err
	}

	p := pkiverifier.NewConfig(caCert.PublicKey.(*ecdsa.PublicKey), caKey, -1)
	token, err := p.CreateTokenFromCertificate(clientCert)
	if err != nil {
		return nil, err
	}
	return token, nil
}
