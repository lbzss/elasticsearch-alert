package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"golang.org/x/xerrors"
)

var (
	GlobalClient *elasticsearch.Client
)

func C() *elasticsearch.Client {
	if GlobalClient == nil {
		panic("Load ClientConfig first to init Elasticsearch Client")
	}
	return GlobalClient
}

type ClientConfig struct {
	TLSEnabled bool   `json:"tls_enabled"`
	CACert     string `json:"ca_cert"`
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	ServerName string `json:"server_name"`
	UserName   string `json:"username"`
	Password   string `json:"password"`
}

func (c *Config) NewESClient() (*elasticsearch.Client, error) {
	var tlsConfig tls.Config
	if !c.Elasticsearch.Client.TLSEnabled {
		tlsConfig.InsecureSkipVerify = true
	} else {
		if c.Elasticsearch.Client.CACert == "" {
			return nil, xerrors.New("no path to CA certificate")
		}
		if c.Elasticsearch.Client.ClientCert == "" {
			return nil, xerrors.New("no path to client certificate")
		}
		if c.Elasticsearch.Client.ClientKey == "" {
			return nil, xerrors.New("no path to client key")
		}
		// Load client certificate
		cert, err := tls.LoadX509KeyPair(c.Elasticsearch.Client.ClientCert, c.Elasticsearch.Client.ClientKey)
		if err != nil {
			return nil, xerrors.Errorf("error loading X509 key pair: %w", err)
		}
		// Load CA certificate
		caCert, err := ioutil.ReadFile(c.Elasticsearch.Client.CACert)
		if err != nil {
			return nil, xerrors.Errorf("error reading CA certificate file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig.RootCAs = caCertPool
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.ServerName = c.Elasticsearch.Client.ServerName
	}
	cfg := elasticsearch.Config{
		Addresses: []string{c.Elasticsearch.Server.ElasticsearchURL},
		Username:  c.Elasticsearch.Client.UserName,
		Password:  c.Elasticsearch.Client.Password,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   30,
			ResponseHeaderTimeout: time.Second * 3,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig: &tlsConfig,
		},
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	return esClient, nil
}
