package tinyraft

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config
type Config struct {
	// node
	NodeID string `env:"TINYRAFT_NODE_ID"`

	DataDir string `env:"TINYRAFT_DATA_DIR"`

	// networking
	ListenAddr    string `env:"TINYRAFT_LISTEN_ADDR"`
	AdvertiseAddr string `env:"TINYRAFT_ADVERTISE_ADDR"`

	// raft
	Bootstrap bool     `env:"TINYRAFT_BOOTSTAP"`
	Join      []string `env:"TINYRAFT_JOIN"`
	Voter     bool     `env:"TINYRAFT_VOTER"`
	TimeoutMS int      `env:"TINYRAFT_TIMEOUT_MS"`
}

// UnmarshalConfigFromEnv
func UnmarshalConfigFromEnv() (*Config, error) {

	defaultTimeoutEnv := os.Getenv("RAFT_TIMEOUT_MS")
	if defaultTimeoutEnv == "" {
		// assign default value is empty
		defaultTimeoutEnv = "3000"
	}

	timeoutMS, err := strconv.Atoi(defaultTimeoutEnv)
	if err != nil {
		return nil, fmt.Errorf("strconv.Atoi RAFT_TIMEOUT_MS: %w", err)
	}

	listenAddr := os.Getenv("NODE_LISTEN_ADDR")
	fmt.Printf("node listen addr: %s\n", listenAddr)
	if listenAddr == "" {
		listenAddr = ":50051"
	}

	advertiseAddr := os.Getenv("NODE_ADVERTISE_ADDR")
	if advertiseAddr == "" {
		advertiseAddr = listenAddr
	}

	return &Config{
		NodeID:        os.Getenv("NODE_ID"),
		Bootstrap:     strings.ToLower(os.Getenv("RAFT_BOOTSTRAP")) == "true",
		Join:          strings.Split(os.Getenv("RAFT_JOIN"), ","),
		Voter:         strings.ToLower(os.Getenv("RAFT_VOTER")) == "true",
		DataDir:       strings.ToLower(os.Getenv("RAFT_DATA_DIR")),
		TimeoutMS:     timeoutMS,
		ListenAddr:    listenAddr,
		AdvertiseAddr: advertiseAddr,
	}, nil
}

func testConfigBootstrap() *Config {
	return &Config{
		NodeID:        "00000000-0000-0000-0000-000000000000",
		Bootstrap:     true,
		Join:          []string{},
		Voter:         true,
		TimeoutMS:     3000,
		DataDir:       "data/1",
		ListenAddr:    "127.0.0.1:50051",
		AdvertiseAddr: "127.0.0.1:50051",
	}
}

func testConfigNode1() *Config {
	return &Config{
		NodeID:        "00000000-0000-0000-0000-00000000001a",
		Bootstrap:     false,
		Join:          []string{"127.0.0.1:50051"},
		Voter:         true,
		TimeoutMS:     3000,
		DataDir:       "data/1",
		ListenAddr:    "127.0.0.1:50052",
		AdvertiseAddr: "127.0.0.1:50052",
	}
}

func testConfigNode2() *Config {
	return &Config{
		NodeID:        "00000000-0000-0000-0000-00000000001b",
		Bootstrap:     false,
		Join:          []string{"127.0.0.1:50051"},
		Voter:         true,
		TimeoutMS:     3000,
		DataDir:       "data/1",
		ListenAddr:    "127.0.0.1:50053",
		AdvertiseAddr: "127.0.0.1:50053",
	}
}
