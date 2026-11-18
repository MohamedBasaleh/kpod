package discover

// Session holds in-memory Later acknowledgements for one process.
type Session struct {
	contexts   map[string]struct{}
	namespaces map[string]map[string]struct{}
}

func NewSession() Session {
	return Session{
		contexts:   map[string]struct{}{},
		namespaces: map[string]map[string]struct{}{},
	}
}

func (s *Session) LaterContext(name string) {
	if s == nil || name == "" {
		return
	}
	if s.contexts == nil {
		s.contexts = map[string]struct{}{}
	}
	s.contexts[name] = struct{}{}
}

func (s *Session) LaterNamespace(kubeContext, namespace string) {
	if s == nil || kubeContext == "" || namespace == "" {
		return
	}
	if s.namespaces == nil {
		s.namespaces = map[string]map[string]struct{}{}
	}
	if s.namespaces[kubeContext] == nil {
		s.namespaces[kubeContext] = map[string]struct{}{}
	}
	s.namespaces[kubeContext][namespace] = struct{}{}
}

func (s Session) ContextDeferred(name string) bool {
	_, ok := s.contexts[name]
	return ok
}

func (s Session) NamespaceDeferred(kubeContext, namespace string) bool {
	_, ok := s.namespaces[kubeContext][namespace]
	return ok
}

// Visible strips session-deferred items without mutating persisted ignore/seen state.
func Visible(r Result, s Session) Result {
	out := cloneResult(r)
	ctxs := make([]string, 0, len(out.NewContexts))
	for _, name := range out.NewContexts {
		if s.ContextDeferred(name) {
			continue
		}
		ctxs = append(ctxs, name)
	}
	out.NewContexts = ctxs
	for ctx, list := range out.NewNamespaces {
		next := make([]string, 0, len(list))
		for _, ns := range list {
			if s.NamespaceDeferred(ctx, ns) {
				continue
			}
			next = append(next, ns)
		}
		if len(next) == 0 {
			delete(out.NewNamespaces, ctx)
		} else {
			out.NewNamespaces[ctx] = next
		}
	}
	return out
}
