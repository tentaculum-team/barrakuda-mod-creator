package cmd

import "testing"

func TestDiffLiveTools(t *testing.T) {
	cases := []struct {
		name     string
		declared []string
		live     []string
		want     []string
	}{
		{"matching sets, no findings", []string{"a", "b"}, []string{"a", "b"}, nil},
		{"live reports an undeclared tool", []string{"a"}, []string{"a", "b"}, []string{`tool "b" is reported by the running server but not declared in the manifest`}},
		{"manifest declares a tool the server never reports", []string{"a", "b"}, []string{"a"}, []string{`tool "b" is declared in the manifest but never reported by the running server`}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := diffLiveTools(c.declared, c.live)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}
