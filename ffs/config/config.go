// Package config defines the configuration settings shared by the
// subcommands of the ffs command-line tool.
package config

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/creachadair/ffs/blob"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/rpcstore"
	yaml "gopkg.in/yaml.v3"
)

// Config represents the stored configuration settings for the ffs tool.
type Config struct {
	// The default address for the blob store service (required).
	StoreAddress string `json:"storeAddress" yaml:"store-address"`
}

// OpenStore connects to the store service address in the configuration.  The
// caller is responsible for closing the store when it is no longer needed.
func (c *Config) OpenStore(_ context.Context) (blob.Store, error) {
	if c.StoreAddress == "" {
		return nil, errors.New("no store service address")
	}
	conn, err := net.Dial(jrpc2.Network(c.StoreAddress))
	if err != nil {
		return nil, fmt.Errorf("dialing store: %w", err)
	}
	ch := channel.Line(conn, conn)
	return rpcstore.NewClient(jrpc2.NewClient(ch, nil), nil), nil
}

// ParseKey parses the string encoding of a key.  By default, s must be hex
// encoded. If s begins with "@", it is taken literally. If s begins with "+"
// it is taken as base64.
func ParseKey(s string) (string, error) {
	if strings.HasPrefix(s, "@") {
		return s[1:], nil
	} else if strings.HasPrefix(s, "+") {
		key, err := base64.StdEncoding.DecodeString(s[1:])
		if err != nil {
			return "", fmt.Errorf("invalid key %q: %w", s, err)
		}
		return string(key), nil
	}
	key, err := hex.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("invalid key %q: %w", s, err)
	}
	return string(key), nil
}

// Load reads and parses the contents of a config file from path.  If the
// specified path does not exist, an empty config is returned without error.
func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return new(Config), nil
	} else if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	cfg := new(Config)
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}
