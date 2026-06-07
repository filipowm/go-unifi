package main

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
)

// modulePath is the generator's import path, used in the DO-NOT-EDIT banner so
// the official surface files match the models file's header form.
const modulePath = "github.com/filipowm/go-unifi/codegen/official"

// doerMethods maps an HTTP verb to the official.Doer method that performs it.
var doerMethods = map[string]string{
	"GET": "Get", "POST": "Post", "PUT": "Put", "DELETE": "Delete", "PATCH": "Patch",
}

// customMethods are the surface methods implemented by hand (info.go, sites.go),
// re-homed onto their resource groups. They carry no generated wrapper body but
// MUST appear in the group interface and its mock, so the generated surface stays
// the single source of truth; the generator never clobbers the hand-written body.
var customMethods = []method{
	{Group: "Info", Name: "Get", Doc: "returns the controller application info (GET /v1/info).", Params: []arg{ctxArg}, Returns: []string{"*Info", "error"}},
	{Group: "Sites", Name: "ListPage", Doc: "returns one page of local sites; nil opts fetches the first page at the default size.", Params: []arg{ctxArg, listOptionsArg}, Returns: []string{"Page[SiteOverview]", "error"}},
	{Group: "Sites", Name: "ListAll", Doc: "lazily drains every local site across pages.", Params: []arg{ctxArg}, Returns: []string{"iter.Seq2[SiteOverview, error]"}},
	{Group: "Sites", Name: "ResolveID", Doc: "maps a legacy site name to its Official-API site UUID, caching the lookup.", Params: []arg{ctxArg, {Name: "name", Type: "string"}}, Returns: []string{"string", "error"}},
}

// ctxArg is the leading context argument shared by every surface method.
var ctxArg = arg{Name: "ctx", Type: "context.Context"}

// listOptionsArg is the page-bounding argument on every ListPage method; its
// runtime type (*ListOptions) is hand-written in unifi/official/pagination.go.
var listOptionsArg = arg{Name: "opts", Type: "*ListOptions"}

// arg is a single method parameter.
type arg struct{ Name, Type string }

// methodKind selects the wrapper-body shape. The zero value (kindScalar) covers
// single-detail/create/update/action wrappers, which dispatch on op shape; list
// operations split into a bounded kindListPage and a lazy kindListAll.
type methodKind int

const (
	kindScalar methodKind = iota
	kindListPage
	kindListAll
)

// method is the unified view a generated wrapper, an interface entry, and a mock
// entry are all rendered from, so the three stay in lockstep.
type method struct {
	Name    string
	Group   string // PascalCase group name; selects the accessor it lives under
	Doc     string // godoc body; the rendered comment is "// <Name> <Doc>"
	Params  []arg
	Returns []string   // Go return types, terminal element is always "error"
	op      *operation // nil for hand-written methods (interface/mock only)
	kind    methodKind
}

// group is one per-tag surface: its accessor/type name plus the methods it carries.
type group struct {
	Name    string
	Methods []method
}

// iface, impl, mock and file derive the Go identifiers and filename for a group.
func (g group) iface() string { return g.Name + "Client" }
func (g group) impl() string  { return lowerFirst(g.Name) + "Client" }
func (g group) mock() string  { return g.Name + "ClientMock" }
func (g group) file() string  { return strings.ToLower(g.Name) + ".generated.go" }

// buildGroups partitions generated operations and hand-written custom methods into
// per-tag groups, sorted for determinism. A stripped method name colliding within
// a group fails loud (mirrors the rename-map collision guard in naming.go).
func buildGroups(ops []operation) ([]group, error) {
	byName := map[string][]method{}
	for _, m := range customMethods {
		byName[m.Group] = append(byName[m.Group], m)
	}
	for i := range ops {
		for _, m := range methodsFor(ops[i]) {
			byName[m.Group] = append(byName[m.Group], m)
		}
	}
	groups := make([]group, 0, len(byName))
	for name, ms := range byName {
		if err := assertNoCollision(name, ms); err != nil {
			return nil, err
		}
		sort.Slice(ms, func(i, j int) bool { return ms[i].Name < ms[j].Name })
		groups = append(groups, group{Name: name, Methods: ms})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups, nil
}

// assertNoCollision fails loud when two operations strip to the same method name.
func assertNoCollision(group string, ms []method) error {
	seen := map[string]bool{}
	for _, m := range ms {
		if seen[m.Name] {
			return fmt.Errorf("group %s: duplicate method name %q after prefix stripping", group, m.Name)
		}
		seen[m.Name] = true
	}
	return nil
}

// methodsFor lifts an operation into the unified method view(s). A list operation
// yields TWO methods — a bounded ListXxxPage and a lazy ListXxxAll — so callers
// opt into draining explicitly; every other shape yields a single method.
func methodsFor(op operation) []method {
	if op.IsList() {
		return []method{listPageMethod(op), listAllMethod(op)}
	}
	return []method{scalarMethod(op)}
}

// baseParams renders the leading parameters shared by every shape: ctx, then the
// path args, then the required query args (all strings).
func baseParams(op operation) []arg {
	params := make([]arg, 0, 1+len(op.PathArgs)+len(op.QueryArgs))
	params = append(params, ctxArg)
	for _, p := range op.PathArgs {
		params = append(params, arg{Name: p.Name, Type: "string"})
	}
	for _, q := range op.QueryArgs {
		params = append(params, arg{Name: q.Name, Type: "string"})
	}
	return params
}

// scalarMethod builds a single-detail/create/update/action wrapper, stripping the
// group prefix from the operationId so the name reads cleanly under its accessor.
func scalarMethod(op operation) method {
	o := op
	m := method{Name: methodName(op), Group: op.Group, Doc: docFor(op), op: &o, kind: kindScalar}
	m.Params = baseParams(op)
	if op.BodyType != "" {
		m.Params = append(m.Params, arg{Name: "body", Type: op.BodyType})
	}
	if op.ReturnType != "" {
		m.Returns = []string{"*" + op.ReturnType, "error"}
	} else {
		m.Returns = []string{"error"}
	}
	return m
}

// listPageMethod builds the bounded ListXxxPage method: one page per call, opts
// govern offset/limit/filter.
func listPageMethod(op operation) method {
	o := op
	m := method{Name: methodName(op) + "Page", Group: op.Group, Doc: pageDoc(op), op: &o, kind: kindListPage}
	m.Params = append(baseParams(op), listOptionsArg)
	m.Returns = []string{"Page[" + op.ItemType + "]", "error"}
	return m
}

// listAllMethod builds the lazy ListXxxAll iterator method: drains every item.
func listAllMethod(op operation) method {
	o := op
	m := method{Name: methodName(op) + "All", Group: op.Group, Doc: allDoc(op), op: &o, kind: kindListAll}
	m.Params = baseParams(op)
	m.Returns = []string{"iter.Seq2[" + op.ItemType + ", error]"}
	return m
}

// docFor builds the godoc body for a scalar operation method.
func docFor(op operation) string {
	return fmt.Sprintf("maps to %s /v1%s on the Official API.", op.HTTPMethod, op.SubPath)
}

// pageDoc/allDoc document the two halves of a list operation's surface.
func pageDoc(op operation) string {
	return fmt.Sprintf("returns one page from %s /v1%s; nil opts fetches the first page at the default size.", op.HTTPMethod, op.SubPath)
}

func allDoc(op operation) string {
	return fmt.Sprintf("lazily drains every item from %s /v1%s, paging on demand; range it and break to stop early.", op.HTTPMethod, op.SubPath)
}

// valueReturn reports whether the method returns a value alongside its error
// (so failure paths must return a leading zero value).
func (m method) valueReturn() bool { return len(m.Returns) > 1 }

// zeroPrefix is the leading return operand(s) on an error path: the first
// return's zero value plus ", " for a value-returning method, empty otherwise.
func (m method) zeroPrefix() string {
	if m.valueReturn() {
		return zeroValue(m.Returns[0]) + ", "
	}
	return ""
}

// zeroValue is the Go zero literal for a return type: nil for pointer/slice/map,
// else a composite-literal (T{}) — so Page[T] returns Page[T]{} on the error path.
func zeroValue(t string) string {
	if strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") {
		return "nil"
	}
	return t + "{}"
}

// signature renders the interface/wrapper signature, e.g.
// "GetNetworkDetails(ctx context.Context, siteId string) (*NetworkDetails, error)".
func (m method) signature() string {
	parts := make([]string, len(m.Params))
	for i, p := range m.Params {
		parts[i] = p.Name + " " + p.Type
	}
	ret := m.Returns[0]
	if len(m.Returns) > 1 {
		ret = "(" + strings.Join(m.Returns, ", ") + ")"
	}
	return fmt.Sprintf("%s(%s) %s", m.Name, strings.Join(parts, ", "), ret)
}

// banner is the shared DO-NOT-EDIT header for every generated surface file.
func banner() string {
	return fmt.Sprintf("// Code generated by %s version %s DO NOT EDIT.", modulePath, generatorVersion)
}

// generateGroupFile renders one group's file: its interface, the *apiClient
// accessor + impl type, the generated wrapper bodies, and the per-group mock.
// Hand-written methods (op == nil) appear in the interface and mock, but their
// bodies live in the hand-written sibling, so they are skipped here.
func generateGroupFile(g group, pkg string) (string, error) {
	var b strings.Builder
	b.WriteString(banner())
	b.WriteString("\n\npackage " + pkg + "\n\n")
	b.WriteString(groupImports(g))

	fmt.Fprintf(&b, "// %s is the %s resource group of the Official UniFi OpenAPI surface.\n", g.iface(), g.Name)
	fmt.Fprintf(&b, "type %s interface {\n", g.iface())
	for _, m := range g.Methods {
		fmt.Fprintf(&b, "\t// %s %s\n\t%s\n", m.Name, m.Doc, m.signature())
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "// %s wraps the shared apiClient so transport, gate and site cache stay single-sourced.\n", g.impl())
	fmt.Fprintf(&b, "type %s struct{ *apiClient }\n\n", g.impl())
	fmt.Fprintf(&b, "var _ %s = %s{}\n\n", g.iface(), g.impl())
	fmt.Fprintf(&b, "// %s returns the %s resource group.\n", g.Name, g.Name)
	fmt.Fprintf(&b, "func (c *apiClient) %s() %s {\n\treturn %s{c}\n}\n", g.Name, g.iface(), g.impl())

	for _, m := range g.Methods {
		if m.op == nil {
			continue
		}
		b.WriteString("\n")
		b.WriteString(wrapperBody(g, m))
	}
	b.WriteString("\n")
	b.WriteString(groupMock(g))
	return formatGo(b.String())
}

// groupImports renders the import block a group file needs: context always, plus
// fmt/net/url/errors only when a generated wrapper body in the group uses them.
func groupImports(g group) string {
	use := groupImportUse(g)
	imports := []string{`"context"`}
	if use["errors"] {
		imports = append(imports, `"errors"`)
	}
	if use["fmt"] {
		imports = append(imports, `"fmt"`)
	}
	if use["iter"] {
		imports = append(imports, `"iter"`)
	}
	if use["net/url"] {
		imports = append(imports, `"net/url"`)
	}
	if len(imports) == 1 {
		return "import " + imports[0] + "\n\n" // context only (hand-written-only group)
	}
	return "import (\n\t" + strings.Join(imports, "\n\t") + "\n)\n\n"
}

// groupImportUse reports which std-lib imports the group file needs. iter is
// driven by signatures (any ListAll iterator return, incl. hand-written ones);
// fmt/net/url/errors are driven by the generated wrapper bodies.
func groupImportUse(g group) map[string]bool {
	use := map[string]bool{}
	for _, m := range g.Methods {
		for _, r := range m.Returns {
			if strings.Contains(r, "iter.Seq2") {
				use["iter"] = true
			}
		}
		if m.op == nil {
			continue
		}
		use["fmt"] = true // every generated wrapper body wraps its error with fmt.
		if len(m.op.QueryArgs) > 0 || len(m.op.PathArgs) > 0 {
			use["net/url"] = true
		}
		if m.op.RequiredFilter() != "" {
			use["errors"] = true
		}
	}
	return use
}

// wrapperBody renders one wrapper method on the group's impl type, dispatching on
// its kind.
func wrapperBody(g group, m method) string {
	switch m.kind {
	case kindListPage:
		return listPageBody(g, m)
	case kindListAll:
		return listAllBody(g, m)
	case kindScalar:
		return scalarBody(g, m)
	default:
		panic(fmt.Sprintf("unknown method kind %d", m.kind))
	}
}

// scalarBody renders a single-detail/create/update/action wrapper: gate check,
// optional required-filter guard, one transport call, wrapped error.
func scalarBody(g group, m method) string {
	op := m.op
	zero := m.zeroPrefix()
	var b strings.Builder
	fmt.Fprintf(&b, "// %s %s\n", m.Name, m.Doc)
	fmt.Fprintf(&b, "func (c %s) %s {\n", g.impl(), m.signature())
	fmt.Fprintf(&b, "\tif err := c.check(ctx); err != nil {\n\t\treturn %serr\n\t}\n", zero)
	if f := op.RequiredFilter(); f != "" {
		fmt.Fprintf(&b, "\tif %s == \"\" {\n\t\treturn %serrors.New(%q)\n\t}\n", f, zero, f+" must not be empty")
	}
	path := pathExpr(*op)
	if op.ReturnType != "" {
		fmt.Fprintf(&b, "\tvar out %s\n", op.ReturnType)
		fmt.Fprintf(&b, "\tif err := c.doer.%s(ctx, c.path(%s), %s, &out); err != nil {\n", doerMethods[op.HTTPMethod], path, reqArg(*op))
		fmt.Fprintf(&b, "\t\treturn nil, fmt.Errorf(%q, err)\n\t}\n", "failed "+m.Name+": %w")
		b.WriteString("\treturn &out, nil\n")
	} else {
		fmt.Fprintf(&b, "\tif err := c.doer.%s(ctx, c.path(%s), %s, nil); err != nil {\n", doerMethods[op.HTTPMethod], path, reqArg(*op))
		fmt.Fprintf(&b, "\t\treturn fmt.Errorf(%q, err)\n\t}\n", "failed "+m.Name+": %w")
		b.WriteString("\treturn nil\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// listPageBody renders the bounded ListXxxPage wrapper: gate check then a single
// page fetch resolved against opts.
func listPageBody(g group, m method) string {
	item := m.op.ItemType
	path := pathExpr(*m.op)
	var b strings.Builder
	fmt.Fprintf(&b, "// %s %s\n", m.Name, m.Doc)
	fmt.Fprintf(&b, "func (c %s) %s {\n", g.impl(), m.signature())
	fmt.Fprintf(&b, "\tif err := c.check(ctx); err != nil {\n\t\treturn Page[%s]{}, err\n\t}\n", item)
	fmt.Fprintf(&b, "\tp, err := listPage[%s](ctx, c.doer, c.path(%s), opts)\n", item, path)
	fmt.Fprintf(&b, "\tif err != nil {\n\t\treturn Page[%s]{}, fmt.Errorf(%q, err)\n\t}\n", item, "failed "+m.Name+": %w")
	b.WriteString("\treturn p, nil\n}\n")
	return b.String()
}

// listAllBody renders the lazy ListXxxAll wrapper: it returns the iterator, which
// runs the gate check and pages on demand when ranged.
func listAllBody(g group, m method) string {
	item := m.op.ItemType
	path := pathExpr(*m.op)
	var b strings.Builder
	fmt.Fprintf(&b, "// %s %s\n", m.Name, m.Doc)
	fmt.Fprintf(&b, "func (c %s) %s {\n", g.impl(), m.signature())
	fmt.Fprintf(&b, "\treturn listSeq[%s](ctx, c.apiClient, c.path(%s), \"\")\n}\n", item, path)
	return b.String()
}

// reqArg is the request-body operand passed to the Doer call.
func reqArg(op operation) string {
	if op.BodyType != "" {
		return "body"
	}
	return "nil"
}

// pathExpr renders the Go expression for the request sub-path: a string literal
// when fully static, else an fmt.Sprintf over the path and required-query args.
func pathExpr(op operation) string {
	if len(op.PathArgs) == 0 && len(op.QueryArgs) == 0 {
		return fmt.Sprintf("%q", op.SubPath)
	}
	var format strings.Builder
	format.WriteString(op.SubPath)
	args := make([]string, 0, len(op.PathArgs)+len(op.QueryArgs))
	// Escape path segments: an ID containing '/', '?' or '#' must not alter the route.
	for _, p := range op.PathArgs {
		args = append(args, "url.PathEscape("+p.Name+")")
	}
	for i, q := range op.QueryArgs {
		sep := "&"
		if i == 0 {
			sep = "?"
		}
		fmt.Fprintf(&format, "%s%s=%%s", sep, q.Name)
		args = append(args, "url.QueryEscape("+q.Name+")")
	}
	return fmt.Sprintf("fmt.Sprintf(%q, %s)", format.String(), strings.Join(args, ", "))
}

// generateClient renders the parent Client interface and its mock: one accessor
// per group, each returning that group's interface. The interface stays the seam
// unifi.Client.Official() returns, so the parent-package wiring is unchanged.
func generateClient(groups []group, pkg string) (string, error) {
	var b strings.Builder
	b.WriteString(banner())
	b.WriteString("\n\npackage " + pkg + "\n\n")
	b.WriteString("// Client is the Official UniFi OpenAPI (integration/v1) surface, exposed as one\n")
	b.WriteString("// fluent accessor per resource group (e.g. Firewall().CreatePolicy(ctx, ...)).\n")
	b.WriteString("type Client interface {\n")
	for _, g := range groups {
		fmt.Fprintf(&b, "\t// %s returns the %s resource group.\n\t%s() %s\n", g.Name, g.Name, g.Name, g.iface())
	}
	b.WriteString("}\n\n")
	b.WriteString("var _ Client = (*apiClient)(nil)\n\n")

	b.WriteString("// ClientMock is a func-field test double implementing Client; each accessor\n")
	b.WriteString("// returns a per-group mock. A nil field panics on call.\n")
	b.WriteString("type ClientMock struct {\n")
	for _, g := range groups {
		fmt.Fprintf(&b, "\t%sFunc func() %s\n", g.Name, g.iface())
	}
	b.WriteString("}\n\n")
	b.WriteString("var _ Client = (*ClientMock)(nil)\n")
	for _, g := range groups {
		fmt.Fprintf(&b, "\nfunc (m *ClientMock) %s() %s {\n\treturn m.%sFunc()\n}\n", g.Name, g.iface(), g.Name)
	}
	return formatGo(b.String())
}

// groupMock renders a hand-rolled func-field mock of one group interface. A bare
// double (no moq dependency) keeps the isolated module's dep graph minimal.
func groupMock(g group) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// %s is a func-field test double implementing %s. A nil field\n", g.mock(), g.iface())
	b.WriteString("// panics on call, surfacing an un-stubbed method in tests.\n")
	fmt.Fprintf(&b, "type %s struct {\n", g.mock())
	for _, m := range g.Methods {
		fmt.Fprintf(&b, "\t%sFunc func(%s) %s\n", m.Name, mockParams(m), mockReturns(m))
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "var _ %s = (*%s)(nil)\n", g.iface(), g.mock())
	for _, m := range g.Methods {
		fmt.Fprintf(&b, "\nfunc (m *%s) %s {\n\treturn m.%sFunc(%s)\n}\n", g.mock(), m.signature(), m.Name, callArgs(m))
	}
	return b.String()
}

// mockParams renders the func-field parameter types (no names).
func mockParams(m method) string {
	types := make([]string, len(m.Params))
	for i, p := range m.Params {
		types[i] = p.Type
	}
	return strings.Join(types, ", ")
}

// mockReturns renders the func-field return clause.
func mockReturns(m method) string {
	if len(m.Returns) > 1 {
		return "(" + strings.Join(m.Returns, ", ") + ")"
	}
	return m.Returns[0]
}

// callArgs renders the argument names forwarded to the stub func, spreading any
// trailing variadic parameter (none today, but kept shape-agnostic).
func callArgs(m method) string {
	names := make([]string, len(m.Params))
	for i, p := range m.Params {
		if strings.HasPrefix(p.Type, "...") {
			names[i] = p.Name + "..."
		} else {
			names[i] = p.Name
		}
	}
	return strings.Join(names, ", ")
}

// formatGo gofmt-formats generated source, surfacing a clear error on bad output.
func formatGo(src string) (string, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", fmt.Errorf("formatting generated source: %w", err)
	}
	return string(out), nil
}
