package config

import "testing"

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("CONFIG_TEST_VALUE", "configured")
	if got := envOrDefault("CONFIG_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("envOrDefault() = %q, want %q", got, "configured")
	}

	t.Setenv("CONFIG_TEST_EMPTY", "")
	if got := envOrDefault("CONFIG_TEST_EMPTY", "fallback"); got != "fallback" {
		t.Fatalf("envOrDefault() = %q, want %q", got, "fallback")
	}
}

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"NATSCONNSTR", "NATSSTREAM", "STORAGECONNSTR", "SERVERPORT", "EXCHANGEADDR", "AUTHSECRET"} {
		t.Setenv(key, "")
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := &Config{
		NatsConnstr:    defaultNatsConnstr,
		NatsStream:     defaultNatsStream,
		StorageConnstr: defaultStorageConnstr,
		ServerAddr:     defaultServerAddr,
		ExchangeAddr:   defaultExchangeAddr,
		AuthSecret:     defaultAuthSecret,
	}
	if *got != *want {
		t.Fatalf("Load() = %+v, want %+v", *got, *want)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	want := &Config{
		NatsConnstr:    "nats://example:4222",
		NatsStream:     "example-stream",
		StorageConnstr: "postgres://example",
		ServerAddr:     "localhost:9090",
		ExchangeAddr:   "exchange:8081",
		AuthSecret:     "test-secret",
	}
	for key, value := range map[string]string{
		"NATSCONNSTR":    want.NatsConnstr,
		"NATSSTREAM":     want.NatsStream,
		"STORAGECONNSTR": want.StorageConnstr,
		"SERVERADDR":     want.ServerAddr,
		"EXCHANGEADDR":   want.ExchangeAddr,
		"AUTHSECRET":     want.AuthSecret,
	} {
		t.Setenv(key, value)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if *got != *want {
		t.Fatalf("Load() = %+v, want %+v", *got, *want)
	}
}
