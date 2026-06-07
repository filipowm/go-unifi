package unifi //nolint: testpackage

import (
	"reflect"
	"testing"
)

// interfacePrivateClientMethods is the explicit allowlist of EXPORTED methods on
// *client that are intentionally NOT part of the public Client interface. Any
// exported *client method outside this set MUST appear in Client — otherwise it
// is unreachable through the public surface (the SetSetting-style drift the
// codegen<->hand-written split repeatedly produces).
//
// Adding a method here is a deliberate decision to keep it interface-private;
// the default is that every exported method is exposed via Client.
var interfacePrivateClientMethods = map[string]string{
	// Takes a ClientInterceptor value; kept off the interface so callers
	// configure interceptors only through ClientConfig at construction time.
	"AddInterceptor": "configured via ClientConfig.Interceptors, not the public interface",
	// Multipart upload helpers are low-level and not part of the curated surface.
	"UploadFile":           "low-level multipart helper; not part of the curated public surface",
	"UploadFileFromReader": "low-level multipart helper; not part of the curated public surface",
	// Raw HTTP transport methods used by official.Doer; not resource-level operations.
	"Patch": "satisfies official.Doer for the Official API layer; not a resource-level InternalClient operation",
}

// TestClientImplementsAllExportedMethods is the reflection drift guard: it walks
// every exported method on *client and asserts each is either declared on the
// Client interface or explicitly allow-listed above. This catches the recurring
// failure where a public method is implemented on the concrete type but never
// wired into the generated Client interface (e.g. SetSetting before O1), leaving
// it uncallable by external consumers of the interface.
func TestClientImplementsAllExportedMethods(t *testing.T) {
	t.Parallel()

	clientType := reflect.TypeFor[*client]()
	ifaceType := reflect.TypeFor[Client]()

	// Collect the set of method names declared on the Client interface.
	ifaceMethods := make(map[string]struct{}, ifaceType.NumMethod())
	for m := range ifaceType.Methods() {
		ifaceMethods[m.Name] = struct{}{}
	}

	for m := range clientType.Methods() {
		// reflect.Type.Method on a non-interface type only returns exported
		// methods, but assert intent explicitly for clarity.
		if !m.IsExported() {
			continue
		}
		name := m.Name
		if _, onIface := ifaceMethods[name]; onIface {
			continue
		}
		if _, allowed := interfacePrivateClientMethods[name]; allowed {
			continue
		}
		t.Errorf("exported *client method %q is not declared on the Client interface and is not "+
			"allow-listed in interfacePrivateClientMethods: it is unreachable through the public "+
			"interface. Either expose it (codegen customizations.yml client.functions) or add it to "+
			"the allowlist with a justification.", name)
	}
}

// TestInterfacePrivateAllowlistIsTight guards the allowlist itself from rotting:
// every entry must correspond to a real exported *client method that is genuinely
// absent from the Client interface. A stale allowlist entry (method removed, or
// later added to the interface) is flagged so the allowlist can't quietly mask a
// future, legitimately-drifted method that happens to share the name.
func TestInterfacePrivateAllowlistIsTight(t *testing.T) {
	t.Parallel()

	clientType := reflect.TypeFor[*client]()
	ifaceType := reflect.TypeFor[Client]()

	for name := range interfacePrivateClientMethods {
		if _, ok := clientType.MethodByName(name); !ok {
			t.Errorf("allowlist entry %q is not an exported method on *client (stale entry)", name)
		}
		if _, ok := ifaceType.MethodByName(name); ok {
			t.Errorf("allowlist entry %q IS on the Client interface; remove it from the allowlist", name)
		}
	}
}
