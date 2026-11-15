package compilermemory

import "encoding/binary"

const (
	jsonObjectFirst uint64 = iota
	jsonObjectKeyRequired
	jsonObjectColon
	jsonObjectValue
	jsonObjectNext
	jsonArrayFirst
	jsonArrayValueRequired
	jsonArrayNext
)

type jsonBuilder struct {
	region            *Region
	input             []byte
	table             []byte
	frames            []byte
	position          int
	count             int
	depth             int
	nodeCap           int
	depthCap          int
	valueDepthLimit   int
	enforceValueDepth bool
}

func (p *jsonBuilder) whitespace() error {
	for p.position < len(p.input) {
		switch p.input[p.position] {
		case ' ', '\t', '\r', '\n':
			p.position++
		default:
			return p.region.actionError()
		}
		if err := p.region.actionError(); err != nil {
			return err
		}
	}
	return p.region.actionError()
}

func (p *jsonBuilder) node(index int) jsonRecord {
	return readJSONRecord(p.table[index*jsonNodeBytes : (index+1)*jsonNodeBytes])
}

func (p *jsonBuilder) emit(kind JSONKind, start, length, parent int) (int, error) {
	if p.count >= p.nodeCap {
		return 0, ErrRange
	}
	index := p.count
	p.count++
	node := jsonRecord{kind: kind, start: uint64(start), length: uint64(length), parent: uint64(parent + 1), end: uint64(p.count)}
	writeJSONRecord(p.table[index*jsonNodeBytes:(index+1)*jsonNodeBytes], node)
	return index, nil
}

func (p *jsonBuilder) state(value uint64) {
	binary.LittleEndian.PutUint64(p.frames[(p.depth-1)*jsonFrameBytes+8:], value)
}

func (p *jsonBuilder) push(index int, state uint64) error {
	if p.depth >= p.depthCap {
		return ErrRange
	}
	frame := p.frames[p.depth*jsonFrameBytes : (p.depth+1)*jsonFrameBytes]
	binary.LittleEndian.PutUint64(frame, uint64(index))
	binary.LittleEndian.PutUint64(frame[8:], state)
	p.depth++
	return nil
}

func (p *jsonBuilder) close(index int) {
	node := p.node(index)
	node.length, node.end = uint64(p.position+1)-node.start, uint64(p.count)
	writeJSONRecord(p.table[index*jsonNodeBytes:(index+1)*jsonNodeBytes], node)
	p.position++
	p.depth--
}

func (p *jsonBuilder) parse() error {
	if err := p.value(-1); err != nil {
		return err
	}
	for p.depth > 0 {
		if err := p.whitespace(); err != nil {
			return err
		}
		if p.position >= len(p.input) {
			return ErrJSON
		}
		frame := p.frames[(p.depth-1)*jsonFrameBytes : p.depth*jsonFrameBytes]
		encodedIndex := binary.LittleEndian.Uint64(frame)
		if encodedIndex >= uint64(p.count) {
			return ErrJSON
		}
		index, err := jsonOffset(encodedIndex, p.count-1)
		if err != nil {
			return err
		}
		state := binary.LittleEndian.Uint64(frame[8:])
		b := p.input[p.position]
		switch state {
		case jsonObjectFirst, jsonObjectKeyRequired:
			if state == jsonObjectFirst && b == '}' {
				p.close(index)
				continue
			}
			if b != '"' {
				return ErrJSON
			}
			start := p.position
			if err := p.stringToken(); err != nil {
				return err
			}
			if err := p.uniqueKey(index, start, p.position-start); err != nil {
				return err
			}
			if _, err := p.emit(jsonObjectKey, start, p.position-start, index); err != nil {
				return err
			}
			p.state(jsonObjectColon)
		case jsonObjectColon:
			if b != ':' {
				return ErrJSON
			}
			p.position++
			p.state(jsonObjectValue)
		case jsonObjectValue:
			p.state(jsonObjectNext)
			if err := p.value(index); err != nil {
				return err
			}
		case jsonObjectNext:
			switch b {
			case '}':
				p.close(index)
			case ',':
				p.position++
				p.state(jsonObjectKeyRequired)
			default:
				return ErrJSON
			}
		case jsonArrayFirst, jsonArrayValueRequired:
			if state == jsonArrayFirst && b == ']' {
				p.close(index)
				continue
			}
			p.state(jsonArrayNext)
			if err := p.value(index); err != nil {
				return err
			}
		case jsonArrayNext:
			switch b {
			case ']':
				p.close(index)
			case ',':
				p.position++
				p.state(jsonArrayValueRequired)
			default:
				return ErrJSON
			}
		default:
			return ErrJSON
		}
	}
	if err := p.whitespace(); err != nil {
		return err
	}
	if p.position != len(p.input) {
		return ErrJSON
	}
	return nil
}

func (p *jsonBuilder) value(parent int) error {
	if p.enforceValueDepth {
		if err := p.region.actionError(); err != nil {
			return err
		}
		if p.depth > p.valueDepthLimit {
			return ErrJSON
		}
	}
	if err := p.whitespace(); err != nil {
		return err
	}
	if p.position >= len(p.input) {
		return ErrJSON
	}
	start := p.position
	b := p.input[p.position]
	switch b {
	case '{', '[':
		kind, state := JSONObject, jsonObjectFirst
		if b == '[' {
			kind, state = JSONArray, jsonArrayFirst
		}
		index, err := p.emit(kind, start, 0, parent)
		if err != nil {
			return err
		}
		p.position++
		return p.push(index, state)
	case '"':
		if err := p.stringToken(); err != nil {
			return err
		}
		_, err := p.emit(JSONString, start, p.position-start, parent)
		return err
	case 't', 'f', 'n':
		word, kind := "true", JSONBoolean
		switch b {
		case 'f':
			word = "false"
		case 'n':
			word, kind = "null", JSONNull
		}
		if len(word) > len(p.input)-p.position {
			return ErrJSON
		}
		for i := range len(word) {
			if p.input[p.position+i] != word[i] {
				return ErrJSON
			}
		}
		p.position += len(word)
		_, err := p.emit(kind, start, len(word), parent)
		return err
	default:
		if err := p.numberToken(); err != nil {
			return err
		}
		_, err := p.emit(JSONNumber, start, p.position-start, parent)
		return err
	}
}

func (p *jsonBuilder) stringToken() error {
	p.position++ // The caller has checked the opening quote.
	for p.position < len(p.input) {
		if err := p.region.actionError(); err != nil {
			return err
		}
		if p.input[p.position] == '"' {
			p.position++
			return nil
		}
		_, next, valid := jsonStringRune(p.input, p.position, len(p.input))
		if !valid {
			return ErrJSON
		}
		p.position = next
	}
	return ErrJSON
}

func (p *jsonBuilder) digits() error {
	start := p.position
	for p.position < len(p.input) && p.input[p.position] >= '0' && p.input[p.position] <= '9' {
		p.position++
		if err := p.region.actionError(); err != nil {
			return err
		}
	}
	if p.position == start {
		return ErrJSON
	}
	return nil
}

func (p *jsonBuilder) numberToken() error {
	if p.input[p.position] == '-' {
		p.position++
	}
	if p.position >= len(p.input) {
		return ErrJSON
	}
	if p.input[p.position] == '0' {
		p.position++
		if p.position < len(p.input) && p.input[p.position] >= '0' && p.input[p.position] <= '9' {
			return ErrJSON
		}
	} else if err := p.digits(); err != nil {
		return err
	}
	if p.position < len(p.input) && p.input[p.position] == '.' {
		p.position++
		if err := p.digits(); err != nil {
			return err
		}
	}
	if p.position < len(p.input) && (p.input[p.position] == 'e' || p.input[p.position] == 'E') {
		p.position++
		if p.position < len(p.input) && (p.input[p.position] == '+' || p.input[p.position] == '-') {
			p.position++
		}
		if err := p.digits(); err != nil {
			return err
		}
	}
	return nil
}

func (p *jsonBuilder) uniqueKey(object, start, length int) error {
	for index := object + 1; index < p.count; {
		if err := p.region.actionError(); err != nil {
			return err
		}
		key := p.node(index)
		if key.kind != jsonObjectKey || index+1 >= p.count {
			return ErrJSON
		}
		keyStart, err := jsonOffset(key.start, len(p.input))
		if err != nil {
			return err
		}
		keyLength, err := jsonOffset(key.length, len(p.input)-keyStart)
		if err != nil {
			return err
		}
		equal, err := p.region.equalJSONStrings(p.input[start:start+length], p.input[keyStart:keyStart+keyLength])
		if err != nil {
			return err
		}
		if equal {
			return ErrJSON
		}
		value := p.node(index + 1)
		if value.end <= uint64(index+1) || value.end > uint64(p.count) {
			return ErrJSON
		}
		index, err = jsonOffset(value.end, p.count)
		if err != nil {
			return err
		}
	}
	return nil
}
