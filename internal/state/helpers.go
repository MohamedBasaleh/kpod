package state

// Clone returns a deep copy of runtime state.
func Clone(s State) State {
	out := s
	out.SeenContexts = append([]string{}, s.SeenContexts...)
	out.SeenNamespaces = make(map[string][]string, len(s.SeenNamespaces))
	for ctx, list := range s.SeenNamespaces {
		out.SeenNamespaces[ctx] = append([]string{}, list...)
	}
	return out
}

func Contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func uniqueAppend(list []string, v string) []string {
	if v == "" || Contains(list, v) {
		return list
	}
	return append(list, v)
}

// MarkContextSeen records that a kubeconfig context has been observed.
func MarkContextSeen(s *State, name string) {
	if s == nil || name == "" {
		return
	}
	s.SeenContexts = uniqueAppend(s.SeenContexts, name)
}

// NamespacesInitialized reports whether this context has a seen-namespace baseline.
func NamespacesInitialized(s State, kubeContext string) bool {
	_, ok := s.SeenNamespaces[kubeContext]
	return ok
}

// MarkNamespaceSeen records that a namespace has been observed in a context.
func MarkNamespaceSeen(s *State, kubeContext, namespace string) {
	if s == nil || kubeContext == "" || namespace == "" {
		return
	}
	if s.SeenNamespaces == nil {
		s.SeenNamespaces = map[string][]string{}
	}
	s.SeenNamespaces[kubeContext] = uniqueAppend(s.SeenNamespaces[kubeContext], namespace)
}

// EnsureNamespaceKey records an empty seen-namespace baseline for a context.
func EnsureNamespaceKey(s *State, kubeContext string) {
	if s == nil || kubeContext == "" {
		return
	}
	if s.SeenNamespaces == nil {
		s.SeenNamespaces = map[string][]string{}
	}
	if _, ok := s.SeenNamespaces[kubeContext]; !ok {
		s.SeenNamespaces[kubeContext] = []string{}
	}
}
