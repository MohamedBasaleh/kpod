package discover

import (
	"fmt"
	"sort"
	"strings"
)

type Kind int

const (
	KindContext Kind = iota
	KindNamespace
)

// Result is the structured discovery diff. The TUI must not compute diffs itself.
type Result struct {
	NewContexts   []string
	NewNamespaces map[string][]string
	Unavailable   []string
	ContextsErr   error
}

// Item is one selectable discovery row.
type Item struct {
	Kind      Kind
	Context   string
	Namespace string
}

func (r Result) Empty() bool {
	if len(r.NewContexts) > 0 {
		return false
	}
	for _, list := range r.NewNamespaces {
		if len(list) > 0 {
			return false
		}
	}
	return true
}

func (r Result) ContextCount() int {
	return len(r.NewContexts)
}

func (r Result) NamespaceCount() int {
	n := 0
	for _, list := range r.NewNamespaces {
		n += len(list)
	}
	return n
}

func (r Result) Count() int {
	return r.ContextCount() + r.NamespaceCount()
}

func (r Result) Badge() string {
	cc, nc := r.ContextCount(), r.NamespaceCount()
	switch {
	case cc > 0 && nc > 0:
		return fmt.Sprintf("%d Kubernetes updates available", cc+nc)
	case cc == 1:
		return "1 new Kubernetes context discovered"
	case cc > 1:
		return fmt.Sprintf("%d new Kubernetes contexts discovered", cc)
	case nc == 1:
		for ctx, list := range r.NewNamespaces {
			if len(list) == 1 {
				return fmt.Sprintf("1 new namespace discovered in %s", ctx)
			}
		}
		return "1 new namespace discovered"
	case nc > 1 && len(r.NewNamespaces) == 1:
		for ctx := range r.NewNamespaces {
			return fmt.Sprintf("%d new namespaces discovered in %s", nc, ctx)
		}
		return fmt.Sprintf("%d new namespaces discovered", nc)
	case nc > 1:
		return fmt.Sprintf("%d new namespaces discovered", nc)
	default:
		return ""
	}
}

func (r Result) Items() []Item {
	items := make([]Item, 0, r.Count())
	ctxs := append([]string{}, r.NewContexts...)
	sort.Strings(ctxs)
	for _, name := range ctxs {
		items = append(items, Item{Kind: KindContext, Context: name})
	}
	nsCtx := make([]string, 0, len(r.NewNamespaces))
	for ctx := range r.NewNamespaces {
		nsCtx = append(nsCtx, ctx)
	}
	sort.Strings(nsCtx)
	for _, ctx := range nsCtx {
		list := append([]string{}, r.NewNamespaces[ctx]...)
		sort.Strings(list)
		for _, ns := range list {
			items = append(items, Item{Kind: KindNamespace, Context: ctx, Namespace: ns})
		}
	}
	return items
}

func (r Result) WithoutContext(name string) Result {
	out := cloneResult(r)
	filtered := make([]string, 0, len(out.NewContexts))
	for _, c := range out.NewContexts {
		if c != name {
			filtered = append(filtered, c)
		}
	}
	out.NewContexts = filtered
	return out
}

func (r Result) WithoutNamespace(kubeContext, namespace string) Result {
	out := cloneResult(r)
	list := out.NewNamespaces[kubeContext]
	if len(list) == 0 {
		return out
	}
	next := make([]string, 0, len(list))
	for _, ns := range list {
		if ns != namespace {
			next = append(next, ns)
		}
	}
	if len(next) == 0 {
		delete(out.NewNamespaces, kubeContext)
	} else {
		out.NewNamespaces[kubeContext] = next
	}
	return out
}

func cloneResult(r Result) Result {
	out := Result{
		NewContexts:   append([]string{}, r.NewContexts...),
		NewNamespaces: map[string][]string{},
		Unavailable:   append([]string{}, r.Unavailable...),
		ContextsErr:   r.ContextsErr,
	}
	for ctx, list := range r.NewNamespaces {
		out.NewNamespaces[ctx] = append([]string{}, list...)
	}
	return out
}

func Format(r Result) string {
	var b strings.Builder
	if r.ContextsErr != nil {
		b.WriteString("Unable to list Kubernetes contexts.\n")
	}
	if r.Empty() && r.ContextsErr == nil {
		return "No new Kubernetes contexts or namespaces.\n"
	}
	if len(r.NewContexts) > 0 {
		b.WriteString("New contexts:\n")
		for _, name := range r.NewContexts {
			fmt.Fprintf(&b, "  %s\n", name)
		}
	}
	if r.NamespaceCount() > 0 {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("New namespaces:\n")
		ctxs := make([]string, 0, len(r.NewNamespaces))
		for ctx := range r.NewNamespaces {
			ctxs = append(ctxs, ctx)
		}
		sort.Strings(ctxs)
		for _, ctx := range ctxs {
			list := append([]string{}, r.NewNamespaces[ctx]...)
			if len(list) == 0 {
				continue
			}
			sort.Strings(list)
			fmt.Fprintf(&b, "  %s:\n", ctx)
			for _, ns := range list {
				fmt.Fprintf(&b, "    %s\n", ns)
			}
		}
	}
	if r.Empty() && r.ContextsErr != nil {
		return b.String()
	}
	if b.Len() == 0 {
		return "No new Kubernetes contexts or namespaces.\n"
	}
	return b.String()
}
