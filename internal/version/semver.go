package version

import (
	"fmt"
	"strconv"
	"strings"
)

type SemVer struct {
	Major, Minor, Patch int
	Pre                 string
	Raw                 string
}

func ParseSemVer(v string) (SemVer, error) {
	raw := strings.TrimSpace(v)
	raw = strings.TrimPrefix(raw, "v")
	if raw == "" {
		return SemVer{}, fmt.Errorf("empty version")
	}
	core, pre, _ := strings.Cut(raw, "-")
	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return SemVer{}, fmt.Errorf("invalid version %q", v)
	}
	nums := make([]int, 3)
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid version %q", v)
		}
		nums[i] = n
	}
	return SemVer{Major: nums[0], Minor: nums[1], Patch: nums[2], Pre: pre, Raw: raw}, nil
}

func (v SemVer) Prerelease() bool {
	return v.Pre != ""
}

// Compare returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b string) (int, error) {
	va, err := ParseSemVer(a)
	if err != nil {
		return 0, err
	}
	vb, err := ParseSemVer(b)
	if err != nil {
		return 0, err
	}
	return va.Compare(vb), nil
}

func (v SemVer) Compare(other SemVer) int {
	if v.Major != other.Major {
		return cmpInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmpInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return cmpInt(v.Patch, other.Patch)
	}
	if v.Pre == other.Pre {
		return 0
	}
	if v.Pre == "" {
		return 1
	}
	if other.Pre == "" {
		return -1
	}
	return comparePre(v.Pre, other.Pre)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func comparePre(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) < n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, aErr := strconv.Atoi(as[i])
		bi, bErr := strconv.Atoi(bs[i])
		if aErr == nil && bErr == nil {
			if c := cmpInt(ai, bi); c != 0 {
				return c
			}
			continue
		}
		if as[i] != bs[i] {
			if as[i] < bs[i] {
				return -1
			}
			return 1
		}
	}
	return cmpInt(len(as), len(bs))
}
