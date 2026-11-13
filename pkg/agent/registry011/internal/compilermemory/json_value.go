package compilermemory

func (v JSONValue) inspectLocked() (jsonRecord, error) {
	d, r := v.document, v.document.owner
	if d.count <= 0 || d.count > r.capacity/jsonNodeBytes || v.index < 0 || v.index >= d.count ||
		d.nodes.length != d.count*jsonNodeBytes || !r.contains(d.input) || !r.contains(d.nodes) || !r.sealed {
		return jsonRecord{}, ErrRange
	}
	start := d.nodes.offset + v.index*jsonNodeBytes
	node := readJSONRecord(r.data[start : start+jsonNodeBytes])
	if node.kind < JSONObject || node.kind > jsonObjectKey || node.start > uint64(d.input.length) ||
		node.length == 0 || node.length > uint64(d.input.length)-node.start ||
		node.end <= uint64(v.index) || node.end > uint64(d.count) || node.parent > uint64(v.index) {
		return jsonRecord{}, ErrJSON
	}
	return node, nil
}

// Raw returns only a checked span, never a native pointer or slice.
func (v JSONValue) Raw() (Span, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return Span{}, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return Span{}, err
	}
	node, err := v.inspectLocked()
	if err != nil {
		return Span{}, err
	}
	if err := r.actionError(); err != nil {
		return Span{}, err
	}
	return v.document.tokenSpan(node)
}

func (v JSONValue) Kind() (JSONKind, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return 0, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return 0, err
	}
	node, err := v.inspectLocked()
	if err != nil {
		return 0, err
	}
	if err := r.actionError(); err != nil {
		return 0, err
	}
	return node.kind, nil
}

// Member searches only direct object members using decoded JSON key equality.
func (v JSONValue) Member(name string) (JSONValue, bool, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return JSONValue{}, false, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return JSONValue{}, false, err
	}
	node, err := v.inspectLocked()
	if err != nil {
		return JSONValue{}, false, err
	}
	if node.kind != JSONObject {
		return JSONValue{}, false, ErrJSONType
	}
	for index := v.index + 1; uint64(index) < node.end; {
		keyValue := JSONValue{document: v.document, index: index}
		key, err := keyValue.inspectLocked()
		if err != nil || key.kind != jsonObjectKey || key.parent != uint64(v.index+1) {
			return JSONValue{}, false, ErrJSON
		}
		keySpan, err := v.document.tokenSpan(key)
		if err != nil {
			return JSONValue{}, false, err
		}
		equal, err := r.equalJSONStringText(r.data[keySpan.offset:keySpan.offset+keySpan.length], name)
		if err != nil {
			return JSONValue{}, false, err
		}
		value := JSONValue{document: v.document, index: index + 1}
		child, err := value.inspectLocked()
		if err != nil || child.parent != uint64(v.index+1) || child.end > node.end || child.kind == jsonObjectKey {
			return JSONValue{}, false, ErrJSON
		}
		if err := r.actionError(); err != nil {
			return JSONValue{}, false, err
		}
		if equal {
			return value, true, nil
		}
		index, err = jsonOffset(child.end, v.document.count)
		if err != nil {
			return JSONValue{}, false, err
		}
	}
	return JSONValue{}, false, r.actionError()
}

func (v JSONValue) Index(wanted int) (JSONValue, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return JSONValue{}, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return JSONValue{}, err
	}
	node, err := v.inspectLocked()
	if err != nil {
		return JSONValue{}, err
	}
	if node.kind != JSONArray {
		return JSONValue{}, ErrJSONType
	}
	if wanted < 0 {
		return JSONValue{}, ErrRange
	}
	for index := v.index + 1; uint64(index) < node.end; wanted-- {
		value := JSONValue{document: v.document, index: index}
		child, err := value.inspectLocked()
		if err != nil || child.parent != uint64(v.index+1) || child.end > node.end || child.kind == jsonObjectKey {
			return JSONValue{}, ErrJSON
		}
		if err := r.actionError(); err != nil {
			return JSONValue{}, err
		}
		if wanted == 0 {
			return value, nil
		}
		index, err = jsonOffset(child.end, v.document.count)
		if err != nil {
			return JSONValue{}, err
		}
	}
	return JSONValue{}, ErrRange
}

func (v JSONValue) scalarSpan(expected JSONKind) (Span, error) {
	kind, err := v.Kind()
	if err != nil {
		return Span{}, err
	}
	if kind != expected {
		return Span{}, ErrJSONType
	}
	return v.Raw()
}

func (v JSONValue) JSONUint64() (uint64, error) {
	s, err := v.scalarSpan(JSONNumber)
	if err != nil {
		return 0, err
	}
	return v.document.owner.JSONUint64(s)
}

func (v JSONValue) JSONInt64() (int64, error) {
	s, err := v.scalarSpan(JSONNumber)
	if err != nil {
		return 0, err
	}
	return v.document.owner.JSONInt64(s)
}

func (v JSONValue) StringEqual(text string) (bool, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return false, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return false, err
	}
	node, err := v.inspectLocked()
	if err != nil {
		return false, err
	}
	if node.kind != JSONString {
		return false, ErrJSONType
	}
	s, err := v.document.tokenSpan(node)
	if err != nil {
		return false, err
	}
	equal, err := r.equalJSONStringText(r.data[s.offset:s.offset+s.length], text)
	if err != nil {
		return false, err
	}
	if err := r.actionError(); err != nil {
		return false, err
	}
	return equal, nil
}

func (v JSONValue) Boolean() (bool, error) {
	s, err := v.scalarSpan(JSONBoolean)
	if err != nil {
		return false, err
	}
	trueValue, err := v.document.owner.Equal(s, []byte("true"))
	if err != nil || trueValue {
		return trueValue, err
	}
	falseValue, err := v.document.owner.Equal(s, []byte("false"))
	if err != nil {
		return false, err
	}
	if !falseValue {
		return false, ErrJSON
	}
	return false, nil
}

// Length counts direct array elements or object members, excluding descendants
// and the native key records. It lets an exact schema reject unknown fields.
func (v JSONValue) Length() (int, error) {
	r := v.document.owner
	if r == nil || r.self != r {
		return 0, ErrConfiguration
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := r.accessError(); err != nil {
		return 0, err
	}
	node, err := v.inspectLocked()
	if err != nil {
		return 0, err
	}
	if node.kind != JSONObject && node.kind != JSONArray {
		return 0, ErrJSONType
	}
	count := 0
	for index := v.index + 1; uint64(index) < node.end; count++ {
		if node.kind == JSONObject {
			key, err := (JSONValue{document: v.document, index: index}).inspectLocked()
			if err != nil || key.kind != jsonObjectKey || key.parent != uint64(v.index+1) {
				return 0, ErrJSON
			}
			index++
		}
		child, err := (JSONValue{document: v.document, index: index}).inspectLocked()
		if err != nil || child.kind == jsonObjectKey || child.parent != uint64(v.index+1) || child.end > node.end {
			return 0, ErrJSON
		}
		if err := r.actionError(); err != nil {
			return 0, err
		}
		index, err = jsonOffset(child.end, v.document.count)
		if err != nil {
			return 0, err
		}
	}
	if err := r.actionError(); err != nil {
		return 0, err
	}
	return count, nil
}
