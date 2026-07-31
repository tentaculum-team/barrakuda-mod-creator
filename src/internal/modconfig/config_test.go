package modconfig

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "flat scalars ok",
			yaml: "api_key: \"\"\nmax_tokens: 4000\nmodo: rapido\n",
		},
		{
			name:    "nested map errors",
			yaml:    "foo:\n  a: 1\n",
			wantErr: true,
		},
		{
			name:    "list value errors",
			yaml:    "foo:\n  - 1\n  - 2\n",
			wantErr: true,
		},
		{
			name:    "malformed yaml errors",
			yaml:    "foo: [unterminated\n",
			wantErr: true,
		},
		{
			name: "empty yaml is an empty config, not an error",
			yaml: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := Parse([]byte(c.yaml))
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil (cfg=%v)", cfg)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParse_ValuesPreserved(t *testing.T) {
	cfg, err := Parse([]byte("api_key: \"\"\nmax_tokens: 4000\nmodo: rapido\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg["api_key"] != "" {
		t.Fatalf("api_key: got %#v, want empty string", cfg["api_key"])
	}
	if cfg["max_tokens"] != 4000 {
		t.Fatalf("max_tokens: got %#v, want 4000", cfg["max_tokens"])
	}
	if cfg["modo"] != "rapido" {
		t.Fatalf("modo: got %#v, want rapido", cfg["modo"])
	}
}
