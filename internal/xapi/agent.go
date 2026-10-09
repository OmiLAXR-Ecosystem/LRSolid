package xapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SGoerzen/LRSolid/internal/store"
)

// ValidateAgentObject checks an actor object, including anonymous groups.
func ValidateAgentObject(agent map[string]any) error {
	return validateAgent(agent, "agent", true)
}

// AgentKey is a stable hash of one agent's inverse functional identifier.
func AgentKey(agent map[string]any) (string, error) {
	canon, err := canonicalIFI(agent)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(sum[:]), nil
}

// ParseAgentFilter parses the xAPI agent query parameter into an IFI.
func ParseAgentFilter(raw string) (store.AgentRef, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return store.AgentRef{}, fmt.Errorf("agent must be a JSON object")
	}
	if err := validateAgent(m, "agent", true); err != nil {
		return store.AgentRef{}, err
	}
	ref, ok := ifiOf(m)
	if !ok {
		return store.AgentRef{}, fmt.Errorf("agent has no inverse functional identifier")
	}
	return ref, nil
}

func canonicalIFI(agent map[string]any) (string, error) {
	ref, ok := ifiOf(agent)
	if ok {
		return canonicalRef(ref), nil
	}
	ot, _ := agent["objectType"].(string)
	if ot == "Group" {
		members, _ := agent["member"].([]any)
		parts := make([]string, 0, len(members))
		for _, member := range members {
			m, _ := member.(map[string]any)
			c, err := canonicalIFI(m)
			if err != nil {
				return "", err
			}
			parts = append(parts, c)
		}
		return "group:" + strings.Join(parts, ","), nil
	}
	return "", fmt.Errorf("agent has no inverse functional identifier")
}

func canonicalRef(ref store.AgentRef) string {
	switch {
	case ref.Mbox != nil:
		return "mbox:" + strings.ToLower(*ref.Mbox)
	case ref.MboxSHA1Sum != nil:
		return "mbox_sha1sum:" + strings.ToLower(*ref.MboxSHA1Sum)
	case ref.OpenID != nil:
		return "openid:" + *ref.OpenID
	case ref.AccountHome != nil || ref.AccountName != nil:
		home, name := "", ""
		if ref.AccountHome != nil {
			home = *ref.AccountHome
		}
		if ref.AccountName != nil {
			name = *ref.AccountName
		}
		return "account:" + home + "|" + name
	default:
		return ""
	}
}

func ifiOf(m map[string]any) (store.AgentRef, bool) {
	var ref store.AgentRef
	n := 0
	if s, ok := stringField(m, "mbox"); ok {
		v := strings.ToLower(s)
		ref.Mbox = &v
		n++
	}
	if s, ok := stringField(m, "mbox_sha1sum"); ok {
		v := strings.ToLower(s)
		ref.MboxSHA1Sum = &v
		n++
	}
	if s, ok := stringField(m, "openid"); ok {
		ref.OpenID = &s
		n++
	}
	if acc, ok := m["account"].(map[string]any); ok {
		home, _ := stringField(acc, "homePage")
		name, _ := stringField(acc, "name")
		if home != "" || name != "" {
			ref.AccountHome = &home
			ref.AccountName = &name
			n++
		}
	}
	return ref, n == 1
}

func stringField(m map[string]any, key string) (string, bool) {
	s, ok := m[key].(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}
