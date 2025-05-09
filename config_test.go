package tinyraft

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
