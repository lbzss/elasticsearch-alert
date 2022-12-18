package utils

import "strings"

func GetAll(json map[string]interface{}, path string) []interface{} {
	raw := getall(0, strings.Split(path, "."), json, "")
	if v, ok := raw.([]interface{}); ok {
		return v
	}
	return []interface{}{raw}
}

func getall(i int, stack []string, elem interface{}, keychain string) interface{} {
	if i > len(stack)-1 {
		if list, ok := elem.([]interface{}); ok {
			var mod []interface{}
			for _, e := range list {
				mod = append(mod, addKey(e, keychain))
			}
			return mod
		}

		if m, ok := elem.(map[string]interface{}); ok {
			return addKey(m, keychain)
		}
		return elem
	}

	key := stack[i]
	if m, ok := elem.(map[string]interface{}); ok {
		v, ok := m[key]
		if !ok {
			return nil
		}
		i++
		return getall(i, stack, v, keychain)
	}

	buckets, ok := elem.([]interface{})
	if !ok {
		return nil
	}

	var mod []interface{}
	for _, item := range buckets {
		kc := keychain
		if e, ok := item.(map[string]interface{}); ok {
			if k, ok := e["key"].(string); ok {
				if kc == "" {
					kc = k
				} else {
					kc = kc + " - " + k
				}
			}
		}

		a := getall(i, stack, item, kc)
		switch v := a.(type) {
		case map[string]interface{}:
			mod = append(mod, v)
		case []interface{}:
			mod = append(mod, v...)
		case nil:
		default:
			mod = append(mod, a)
		}

	}

	return mod
}

func addKey(i interface{}, keychain string) interface{} {
	obj, ok := i.(map[string]interface{})
	if !ok {
		return i
	}
	key, ok := obj["key"].(string)
	if !ok {
		return obj
	}
	if key == "" {
		return obj
	}
	if keychain != "" {
		obj["key"] = keychain + "-" + key
	}
	return obj
}
