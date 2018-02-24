package storage

import (
	"testing"
)

func TestConnStr(t *testing.T) {
	tc := func(cfg PostgresConfig, connstr string) {
		t.Helper()
		actual := cfg.connStr()
		if actual != connstr {
			t.Errorf("Expected %#v => \n %q, got\n %q", cfg, connstr, actual)
		}
	}

	tc(PostgresConfig{Host: "example.com", DBName: "testdb", User: "testuser", Password: "testpass", Port: "1234"},
		"host=example.com port=1234 sslmode=disable dbname=testdb user=testuser password=testpass")
	tc(PostgresConfig{Host: "example.com", DBName: "testdb"}, "host=example.com sslmode=disable dbname=testdb")

}
