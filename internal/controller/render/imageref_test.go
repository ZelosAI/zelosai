package render

import "testing"

func TestImageRef(t *testing.T) {
	cases := []struct {
		name, repo, tagOrDigest, want string
	}{
		{"ordinary tag", "ghcr.io/zelosai/zelosgateway", "develop", "ghcr.io/zelosai/zelosgateway:develop"},
		{"semver tag", "harbor.svc/zelos/zelosmcp", "v0.3.0", "harbor.svc/zelos/zelosmcp:v0.3.0"},
		{"digest", "harbor.svc/zelos/zelosgateway", "sha256:abc123", "harbor.svc/zelos/zelosgateway@sha256:abc123"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ImageRef(c.repo, c.tagOrDigest); got != c.want {
				t.Fatalf("ImageRef(%q,%q) = %q, want %q", c.repo, c.tagOrDigest, got, c.want)
			}
		})
	}
}
