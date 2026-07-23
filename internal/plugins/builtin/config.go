package builtin

func cfgInt(cfg map[string]any, key string, def int) int {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return def
}
