package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

var (
	ErrMissingOption     = errors.New("missing option")
	ErrMissingAuthMethod = errors.New("missing auth method, must provide either password or private key")
)

func Load() (*Config, error) {
	cfg := &Config{}

	flag.BoolVar(&cfg.DevMode, "dev", false, "development mode, reloads web assets on every request")
	flag.StringVar(&cfg.ListenAddr, "listen", "localhost:8080", "http listening address as <ip>[:<port>]")
	flag.StringVar(&cfg.DeviceAddr, "device", "", "device address as <ip>[:<port>]")
	flag.StringVar(&cfg.Username, "user", "", "username for SSH authentication")
	flag.StringVar(&cfg.Password, "pass", "", "password for SSH authentication (public key recommended)")
	flag.StringVar(&cfg.PrivateKeyPath, "pk", "", "path of private key for SSH authentication (default is ~/.ssh/id_rsa)")
	flag.Parse()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

type Config struct {
	DevMode        bool
	ListenAddr     string
	DeviceAddr     string
	Username       string
	Password       string
	PrivateKeyPath string
}

func (c *Config) Help(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
	flag.PrintDefaults()
}

func (c *Config) validate() error {
	if c.DeviceAddr == "" {
		return fmt.Errorf("%w: %s", ErrMissingOption, "device")
	}

	var (
		host string
		port string
		err  error
	)

	host, port, err = net.SplitHostPort(c.DeviceAddr)
	if err != nil {
		// no port was provided, default to 22
		port = "22"

		ip := net.ParseIP(c.DeviceAddr)
		if ip == nil {
			return fmt.Errorf("invalid address: %q", c.DeviceAddr)
		}

		host = ip.String()
	}
	c.DeviceAddr = net.JoinHostPort(host, port)

	if c.Username == "" {
		return fmt.Errorf("%w: %s", ErrMissingOption, "user")
	}

	return nil
}

func (c *Config) ToSSHConfig() (*ssh.ClientConfig, error) {
	auth := []ssh.AuthMethod{}

	if defaultSigner := defaultPrivateKey(); defaultSigner != nil {
		auth = append(auth, ssh.PublicKeys(defaultSigner))
	} else if c.PrivateKeyPath != "" {
		signer, err := readPrivateKey(c.PrivateKeyPath)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if c.Password != "" {
		auth = append(auth, ssh.Password(c.Password))
	}

	if len(auth) == 0 {
		return nil, ErrMissingAuthMethod
	}

	return &ssh.ClientConfig{
		User:            c.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: host key changes on switch restart??
	}, nil
}

func readPrivateKey(file string) (ssh.Signer, error) {
	key, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(key)
}

func defaultPrivateKey() ssh.Signer {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// afaik unifi only supports rsa keys
	file := filepath.Join(home, ".ssh", "id_rsa")
	signer, err := readPrivateKey(file)
	if err != nil {
		return nil
	}

	return signer
}
