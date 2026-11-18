package shortcut

import (
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
)

var wellKnownEnv = map[string]string{
	"dev":         "d",
	"development": "d",
	"staging":     "s",
	"stage":       "s",
	"production":  "p",
	"prod":        "p",
	"preview":     "pv",
}

func actionSuffix(id action.ID) string {
	spec, err := action.Resolve(string(id))
	if err == nil && spec.Shortcut != "" {
		return spec.Shortcut
	}
	switch id {
	case action.DBConsole:
		return "db"
	case action.Runner:
		return "r"
	case action.PortForward:
		return "pf"
	default:
		s := strings.ToLower(string(id))
		if s == "" {
			return "x"
		}
		return s[:1]
	}
}

func EnvPrefix(alias string, all []string) string {
	alias = strings.ToLower(strings.TrimSpace(alias))
	if alias == "" {
		return "x"
	}
	if p, ok := wellKnownEnv[alias]; ok && uniqueAmong(p, alias, all, func(other string) string {
		o := strings.ToLower(other)
		if v, ok := wellKnownEnv[o]; ok {
			return v
		}
		if o == "" {
			return ""
		}
		return o[:1]
	}) {
		return p
	}
	for n := 1; n <= len([]rune(alias)); n++ {
		p := string([]rune(alias)[:n])
		clash := false
		for _, other := range all {
			o := strings.ToLower(other)
			if o == alias {
				continue
			}
			or := []rune(o)
			if len(or) >= n && string(or[:n]) == p {
				clash = true
				break
			}
		}
		if !clash {
			return p
		}
	}
	return alias
}

func uniqueAmong(prefix, alias string, all []string, of func(string) string) bool {
	for _, other := range all {
		if strings.ToLower(other) == alias {
			continue
		}
		if of(other) == prefix {
			return false
		}
	}
	return true
}

func Suggest(env string, act action.ID, cfg config.Config, extraTaken map[string]struct{}) string {
	all := cfg.Aliases()
	prefix := EnvPrefix(env, all)
	base := prefix + actionSuffix(act)
	if !taken(base, cfg, extraTaken) {
		return base
	}
	for n := len(prefix) + 1; n <= len(env)+len(actionSuffix(act))+4; n++ {
		cand := env
		if n < len(env) {
			cand = env[:n]
		}
		id := cand + actionSuffix(act)
		if !taken(id, cfg, extraTaken) {
			return id
		}
	}
	i := 2
	for {
		id := base + itoa(i)
		if !taken(id, cfg, extraTaken) {
			return id
		}
		i++
	}
}

func taken(id string, cfg config.Config, extra map[string]struct{}) bool {
	if _, ok := cfg.Shortcuts[id]; ok {
		return true
	}
	if extra != nil {
		if _, ok := extra[id]; ok {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [12]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}

type Candidate struct {
	ID        string
	Env       string
	Action    action.ID
	DefaultOn bool
}

func Recommend(cfg config.Config) []Candidate {
	envs := cfg.Aliases()
	taken := map[string]struct{}{}
	var out []Candidate
	type rec struct {
		act action.ID
		on  bool
	}
	for _, env := range envs {
		for _, r := range []rec{{action.Console, true}, {action.Shell, true}, {action.Logs, false}} {
			id := Suggest(env, r.act, cfg, taken)
			taken[id] = struct{}{}
			out = append(out, Candidate{ID: id, Env: env, Action: r.act, DefaultOn: r.on && !rOnDanger(r.act)})
		}
	}
	return out
}

func rOnDanger(id action.ID) bool {
	spec, err := action.Resolve(string(id))
	return err == nil && spec.Destructive
}
