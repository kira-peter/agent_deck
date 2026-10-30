package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// orderedObject is a JSON object that remembers key order and keeps every
// value as raw JSON, so editing one key never disturbs the others.
type orderedObject struct {
	keys []string
	vals map[string]json.RawMessage
}

func newOrderedObject() *orderedObject {
	return &orderedObject{vals: map[string]json.RawMessage{}}
}

func parseOrderedObject(data []byte) (*orderedObject, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not a JSON object")
	}

	o := newOrderedObject()
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := kt.(string)
		if !ok {
			return nil, errors.New("invalid object key")
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		o.set(key, raw)
	}
	if _, err := dec.Token(); err != nil { // closing brace
		return nil, err
	}
	return o, nil
}

func (o *orderedObject) set(key string, raw json.RawMessage) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = raw
}

func (o *orderedObject) unset(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

func (o *orderedObject) marshal() json.RawMessage {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := encodeJSON(k)
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(o.vals[k])
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

func (o *orderedObject) lookup(parts []string) (json.RawMessage, bool) {
	raw, ok := o.vals[parts[0]]
	if !ok {
		return nil, false
	}
	if len(parts) == 1 {
		return raw, true
	}
	child, err := parseOrderedObject(raw)
	if err != nil {
		return nil, false
	}
	return child.lookup(parts[1:])
}

func (o *orderedObject) setPath(parts []string, v json.RawMessage) error {
	if len(parts) == 1 {
		o.set(parts[0], v)
		return nil
	}
	child := newOrderedObject()
	if raw, ok := o.vals[parts[0]]; ok {
		c, err := parseOrderedObject(raw)
		if err != nil {
			return fmt.Errorf("%s is not an object: %w", parts[0], err)
		}
		child = c
	}
	if err := child.setPath(parts[1:], v); err != nil {
		return err
	}
	o.set(parts[0], child.marshal())
	return nil
}

func (o *orderedObject) unsetPath(parts []string) error {
	if len(parts) == 1 {
		o.unset(parts[0])
		return nil
	}
	raw, ok := o.vals[parts[0]]
	if !ok {
		return nil
	}
	child, err := parseOrderedObject(raw)
	if err != nil {
		return fmt.Errorf("%s is not an object: %w", parts[0], err)
	}
	if err := child.unsetPath(parts[1:]); err != nil {
		return err
	}
	o.set(parts[0], child.marshal())
	return nil
}

// encodeJSON marshals v without escaping <, > and &.
func encodeJSON(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// SettingEntry is one key/value pair of a settings object.
type SettingEntry struct {
	Key   string
	Value json.RawMessage
}

// ClaudeSettings is Claude Code's settings.json. Edits keep key order and
// every key this tool does not know about.
type ClaudeSettings struct {
	Path string
	root *orderedObject
}

// LoadClaudeSettings reads settings.json. A missing or empty file yields empty
// settings; a file that is not a JSON object is an error.
func LoadClaudeSettings(path string) (*ClaudeSettings, error) {
	s := &ClaudeSettings{Path: path, root: newOrderedObject()}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(data)) == 0 {
		return s, nil
	}

	root, err := parseOrderedObject(data)
	if err != nil {
		return nil, err
	}
	s.root = root
	return s, nil
}

// Keys returns the top-level keys in file order.
func (s *ClaudeSettings) Keys() []string {
	return append([]string(nil), s.root.keys...)
}

// Lookup returns the raw value at a dotted path such as "permissions.defaultMode".
func (s *ClaudeSettings) Lookup(path string) (json.RawMessage, bool) {
	return s.root.lookup(strings.Split(path, "."))
}

// Entries returns the key/value pairs of the object at path; an empty path
// means the top level.
func (s *ClaudeSettings) Entries(path string) []SettingEntry {
	obj := s.root
	if path != "" {
		raw, ok := s.Lookup(path)
		if !ok {
			return nil
		}
		var err error
		if obj, err = parseOrderedObject(raw); err != nil {
			return nil
		}
	}
	entries := make([]SettingEntry, 0, len(obj.keys))
	for _, k := range obj.keys {
		entries = append(entries, SettingEntry{Key: k, Value: obj.vals[k]})
	}
	return entries
}

// StringList returns the string array at path, or nil.
func (s *ClaudeSettings) StringList(path string) []string {
	raw, ok := s.Lookup(path)
	if !ok {
		return nil
	}
	var list []string
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	return list
}

// Set stores value at a dotted path, creating parent objects as needed.
func (s *ClaudeSettings) Set(path string, value any) error {
	raw, err := encodeJSON(value)
	if err != nil {
		return err
	}
	return s.root.setPath(strings.Split(path, "."), raw)
}

// Unset removes the key at a dotted path.
func (s *ClaudeSettings) Unset(path string) error {
	return s.root.unsetPath(strings.Split(path, "."))
}

// Save writes the file with 2-space indentation, via a temp file so a crash
// cannot leave a half-written settings.json.
func (s *ClaudeSettings) Save() error {
	var out bytes.Buffer
	if err := json.Indent(&out, s.root.marshal(), "", "  "); err != nil {
		return err
	}
	out.WriteByte('\n')

	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, out.Bytes(), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
