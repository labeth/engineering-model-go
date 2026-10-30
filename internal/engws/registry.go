// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
)

// Publication is the registry answer for one module version.
type Publication int

const (
	// PublicationUnknown means the registry could not be asked (offline,
	// unconfigured or failing); it never marks a checkout changed.
	PublicationUnknown Publication = iota
	PublicationPublished
	PublicationMissing
)

// Registry answers whether a module version is published.
type Registry interface {
	// Published reports the publication of modulePath (with @vN) at version
	// and, for PublicationUnknown, why.
	Published(ctx context.Context, modulePath, version string) (Publication, string)
}

// cueRegistry resolves module paths with CUE's own configuration
// (CUE_REGISTRY prefix routing, +insecure, none, logins.json and credential
// helpers) and lists tags through the OCI distribution API.
type cueRegistry struct {
	env      string
	resolver *modconfig.Resolver
	client   *modregistry.Client
	initErr  error

	mu       sync.Mutex
	versions map[string][]string
	failures map[string]string
}

// NewCUERegistry returns the registry configured by CUE_REGISTRY. An unset
// CUE_REGISTRY is reported as unknown instead of falling back to CUE's
// public default registry, so engws never contacts a registry the project
// did not configure.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func NewCUERegistry() Registry {
	r := &cueRegistry{env: strings.TrimSpace(os.Getenv("CUE_REGISTRY")), versions: map[string][]string{}, failures: map[string]string{}}
	if r.env == "" {
		return r
	}
	resolver, err := modconfig.NewResolver(nil)
	if err != nil {
		r.initErr = err
		return r
	}
	r.resolver = resolver
	r.client = modregistry.NewClientWithResolver(resolver)
	return r
}

// Published lists the module's tags once per module path and looks the
// version up. The repository is the module path without @vN, as CUE maps it.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func (r *cueRegistry) Published(ctx context.Context, modulePath, version string) (Publication, string) {
	if r.env == "" {
		return PublicationUnknown, "CUE_REGISTRY is not set; publication not checked"
	}
	if r.initErr != nil {
		return PublicationUnknown, fmt.Sprintf("CUE_REGISTRY: %v", r.initErr)
	}
	base, _, _ := strings.Cut(modulePath, "@")
	if _, ok := r.resolver.ResolveToLocation(base, ""); !ok {
		return PublicationUnknown, fmt.Sprintf("CUE_REGISTRY routes %s to no registry (none/offline)", base)
	}
	r.mu.Lock()
	versions, cached := r.versions[modulePath]
	failure := r.failures[modulePath]
	r.mu.Unlock()
	if !cached && failure == "" {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		list, err := r.client.ModuleVersions(ctx, modulePath)
		r.mu.Lock()
		if err != nil {
			failure = fmt.Sprintf("registry query for %s failed: %v", modulePath, err)
			r.failures[modulePath] = failure
		} else {
			versions = list
			r.versions[modulePath] = list
		}
		r.mu.Unlock()
	}
	if failure != "" {
		return PublicationUnknown, failure
	}
	if slices.Contains(versions, tagFor(version)) {
		return PublicationPublished, ""
	}
	return PublicationMissing, ""
}
