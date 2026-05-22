// Package render builds Kubernetes objects honoring the Zelos suite container
// contract defined in docs/architecture/07-container-contract.md.
package render

import (
	"fmt"
	"strings"
)

// Component captures the per-component constants used by every render builder.
// One Component is defined per Zelos workload. See contract.go for the table.
type Component struct {
	// Name is the short component name used in labels, env var prefix, and resource names.
	// Example: "zelosgateway".
	Name string

	// EnvPrefix is the uppercase per-component env var prefix (default: uppercase Name).
	EnvPrefix string

	// Port is the primary HTTP port the container exposes.
	Port int32

	// DefaultImageRepo is "ghcr.io/zelosai/<Name>" by default.
	DefaultImageRepo string

	// PersistentStateDir is where the workload writes durable state.
	// Defaults to "/var/lib/zelos/<Name>".
	PersistentStateDir string

	// SecretsDir is the standard secret-file mount root ("/etc/zelos/secrets").
	SecretsDir string

	// TLSDir is the standard TLS mount root ("/etc/zelos/tls").
	TLSDir string
}

// Defaults returns a Component populated with the suite-wide defaults for the named component.
func Defaults(name string) Component {
	c := Component{
		Name:               name,
		EnvPrefix:          strings.ToUpper(name),
		DefaultImageRepo:   fmt.Sprintf("ghcr.io/zelosai/%s", name),
		PersistentStateDir: fmt.Sprintf("/var/lib/zelos/%s", name),
		SecretsDir:         "/etc/zelos/secrets",
		TLSDir:             "/etc/zelos/tls",
	}
	switch name {
	case "zelosbroker":
		c.Port = 8080
	case "zelosbackplane":
		c.Port = 8080
	default: // zelosgateway, zelosmcp, zelosserver
		c.Port = 8000
	}
	return c
}

// Labels returns the standard label set applied to every object the operator creates.
func Labels(c Component, instance string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       c.Name,
		"app.kubernetes.io/instance":   instance,
		"app.kubernetes.io/managed-by": "zelosai",
		"app.kubernetes.io/part-of":    "zelos",
		"zelos.zelosai.io/component":   c.Name,
	}
}

// SelectorLabels returns the subset used in Service selectors and Deployment matchLabels.
func SelectorLabels(c Component, instance string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     c.Name,
		"app.kubernetes.io/instance": instance,
	}
}

// FileEnvName is the standard env var name pointing at a Secret-mounted file.
func FileEnvName(c Component, key string) string {
	return fmt.Sprintf("%s_%s_FILE", c.EnvPrefix, strings.ToUpper(strings.ReplaceAll(key, "-", "_")))
}
