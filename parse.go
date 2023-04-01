package sf

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

var (
	// ErrUnexpectedEOL reports an unexpected end-of-line.
	ErrUnexpectedEOL = errors.New("sf: unexpected EOL")

	// ErrUnrecognized reports an illegal character.
	ErrUnrecognized = errors.New("sf: unrecognized char")

	// ErrTooManyDigits reports a too big integer or fractional component.
	ErrTooManyDigits = errors.New("sf: too many digits")
)

// Parse parses an structured-field dictionary multi-line header.
func ParseDict(header []string) (Dict, error) {
	return ParseDictLine(joinMultiLines(header))
}

// ParseLine parses a structured-field dictionary single-line header.
func ParseDictLine(header string) (Dict, error) {
	var (
		dict  Dict
		pair  *Pair
		err   error
		input = []byte(header)
		pos   = 0
	)
	pos = skipSpaces(input, pos)
	if pos >= len(input) {
		return dict, nil
	}
	for {
		pair, pos, err = parsePair(input, pos)
		if err != nil {
			return nil, err
		}
		dict = dict.Add(pair.Key, pair.Value)
		pos = skipSpaces(input, pos)
		if pos >= len(input) || input[pos] != ',' {
			break
		}
		pos++
	}
	return dict, nil
}

// Parse parses an structured-field list multi-line header.
func ParseList(header []string) (List, error) {
	return ParseListLine(joinMultiLines(header))
}

// ParseLine parses a structured-field list single-line header.
func ParseListLine(header string) (List, error) {
	var (
		list   List
		member Member
		err    error
		input  = []byte(header)
		pos    = 0
	)
	pos = skipSpaces(input, pos)
	if pos >= len(input) {
		return list, nil
	}
	for {
		member, pos, err = parseMember(input, pos)
		if err != nil {
			return nil, err
		}
		list = append(list, member)
		pos = skipSpaces(input, pos)
		if pos >= len(input) || input[pos] != ',' {
			break
		}
		pos++
	}
	return list, nil
}

// ParseLine parses a structured-field item single-line header.
func ParseItemLine(header string) (*Item, error) {
	input := []byte(header)
	it, pos, err := parseItem(input, 0)
	if err != nil {
		return nil, err
	}
	pos = skipSpaces(input, pos)
	if pos < len(input) {
		return nil, ErrUnrecognized
	}
	return it, nil
}

func joinMultiLines(header []string) string {
	nonEmptyLines := make([]string, 0, len(header))
	for _, h := range header {
		if trimmed := strings.TrimSpace(h); trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
		}
	}
	return strings.Join(nonEmptyLines, ", ")
}

func parsePair(input []byte, pos int) (*Pair, int, error) {
	key, pos, err := parseKey(input, pos)
	if err != nil {
		return nil, pos, err
	}
	var (
		value Member
		p     ParamList
	)
	if pos < len(input) && input[pos] == '=' {
		value, pos, err = parseMember(input, pos+1)
		if err != nil {
			return nil, pos, err
		}
	} else {
		p, pos, err = parseParams(input, pos)
		if err != nil {
			return nil, pos, err
		}
		value = &Item{Bool(true), p}
	}
	return &Pair{key, value}, pos, nil
}

func parseMember(input []byte, pos int) (Member, int, error) {
	pos = skipSpaces(input, pos)
	if pos >= len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	if input[pos] == '(' {
		return parseInnerList(input, pos)
	}
	return parseItem(input, pos)
}

func parseInnerList(input []byte, pos int) (*InnerList, int, error) {
	if pos >= len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	if input[pos] != '(' {
		return nil, pos, ErrUnrecognized
	}
	pos++
	var (
		items []Item
		it    *Item
		err   error
	)
	for {
		pos = skipSpaces(input, pos)
		if pos >= len(input) {
			return nil, pos, ErrUnexpectedEOL
		}
		if input[pos] == ')' {
			pos++
			break
		}
		it, pos, err = parseItem(input, pos)
		if err != nil {
			return nil, pos, err
		}
		items = append(items, *it)
	}
	p, pos, err := parseParams(input, pos)
	if err != nil {
		return nil, pos, err
	}
	return &InnerList{items, p}, pos, nil
}

func parseItem(input []byte, pos int) (*Item, int, error) {
	b, pos, err := parseBareItem(input, pos)
	if err != nil {
		return nil, pos, err
	}
	p, pos, err := parseParams(input, pos)
	if err != nil {
		return nil, pos, err
	}
	return &Item{b, p}, pos, nil
}

func parseParams(input []byte, pos int) (ParamList, int, error) {
	var (
		params ParamList
		key    string
		value  BareItem
		err    error
	)
	for {
		pos = skipSpaces(input, pos)
		if pos >= len(input) || input[pos] != ';' {
			break
		}
		key, pos, err = parseKey(input, pos+1)
		if err != nil {
			return nil, pos, err
		}
		value = Bool(true)
		if pos < len(input) && input[pos] == '=' {
			value, pos, err = parseBareItem(input, pos+1)
			if err != nil {
				return nil, pos, err
			}
		}
		params = params.Add(key, value)
	}
	return params, pos, nil
}

func parseKey(input []byte, pos int) (string, int, error) {
	pos = skipSpaces(input, pos)
	if pos >= len(input) {
		return "", pos, ErrUnexpectedEOL
	}
	if input[pos] != '*' && !isLower(input[pos]) {
		return "", pos, ErrUnrecognized
	}
	var sb strings.Builder
	for pos < len(input) && isKeyChar(input[pos]) {
		sb.WriteByte(input[pos])
		pos++
	}
	return sb.String(), pos, nil
}

var bareItemParsers = []struct {
	Cond  func(byte) bool
	Parse func([]byte, int) (BareItem, int, error)
}{
	{func(b byte) bool { return b == '-' || isDigit(b) }, parseNumber},
	{func(b byte) bool { return b == '"' }, parseString},
	{func(b byte) bool { return b == '*' || isAlpha(b) }, parseToken},
	{func(b byte) bool { return b == ':' }, parseByteSeq},
	{func(b byte) bool { return b == '?' }, parseBool},
}

func parseBareItem(input []byte, pos int) (BareItem, int, error) {
	pos = skipSpaces(input, pos)
	if pos >= len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	for _, p := range bareItemParsers {
		if p.Cond(input[pos]) {
			return p.Parse(input, pos)
		}
	}
	return nil, pos, ErrUnrecognized
}

func parseNumber(input []byte, pos int) (BareItem, int, error) {
	if input[pos] != '-' && !isDigit(input[pos]) {
		return nil, pos, ErrUnrecognized
	}
	sign := int64(1)
	if input[pos] == '-' {
		sign = -1
		pos++
	}
	if pos == len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	if !isDigit(input[pos]) {
		return nil, pos, ErrUnrecognized
	}
	var sb strings.Builder
	decimalPlaces := -1
	for pos < len(input) {
		if isDigit(input[pos]) {
			if sb.Len() == 15 {
				return nil, pos, ErrTooManyDigits
			}
			sb.WriteByte(input[pos])
			if decimalPlaces >= 0 {
				if decimalPlaces == 3 {
					return nil, pos, ErrTooManyDigits
				}
				decimalPlaces++
			}
		} else if input[pos] == '.' {
			if decimalPlaces != -1 {
				break
			}
			decimalPlaces = 0
		} else {
			break
		}
		pos++
	}
	n, _ := strconv.ParseInt(sb.String(), 10, 64)
	switch decimalPlaces {
	case -1:
		return Integer(sign * n), pos, nil
	case 1:
		return Decimal(sign * n * 100), pos, nil
	case 2:
		return Decimal(sign * n * 10), pos, nil
	case 3:
		return Decimal(sign * n), pos, nil
	}
	return nil, pos, ErrUnrecognized
}

func parseString(input []byte, pos int) (BareItem, int, error) {
	if input[pos] != '"' {
		return nil, pos, ErrUnrecognized
	}
	var sb strings.Builder
	sb.WriteByte(input[pos])
	pos++
	for pos < len(input) && input[pos] != '"' {
		if input[pos] == '\\' {
			sb.WriteByte(input[pos])
			pos++
			if pos == len(input) {
				return nil, pos, ErrUnexpectedEOL
			}
			if input[pos] != '"' && input[pos] != '\\' {
				return nil, pos, ErrUnrecognized
			}
		}
		if !isPrint(input[pos]) {
			return nil, pos, ErrUnrecognized
		}
		sb.WriteByte(input[pos])
		pos++
	}
	if pos == len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	sb.WriteByte(input[pos])
	s, _ := strconv.Unquote(sb.String())
	return String(s), pos + 1, nil
}

func parseToken(input []byte, pos int) (BareItem, int, error) {
	if input[pos] != '*' && !isAlpha(input[pos]) {
		return nil, pos, ErrUnrecognized
	}
	var sb strings.Builder
	for pos < len(input) && isTokenChar(input[pos]) {
		sb.WriteByte(input[pos])
		pos++
	}
	return Token(sb.String()), pos, nil
}

func parseByteSeq(input []byte, pos int) (BareItem, int, error) {
	if input[pos] != ':' {
		return nil, pos, ErrUnrecognized
	}
	var sb strings.Builder
	pos++
	for pos < len(input) && input[pos] != ':' {
		if !isBase64Char(input[pos]) {
			return nil, pos, ErrUnrecognized
		}
		sb.WriteByte(input[pos])
		pos++
	}
	if pos == len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	b, _ := base64.StdEncoding.DecodeString(sb.String())
	return ByteSeq(b), pos + 1, nil
}

func parseBool(input []byte, pos int) (BareItem, int, error) {
	if input[pos] != '?' {
		return nil, pos, ErrUnrecognized
	}
	pos++
	if pos == len(input) {
		return nil, pos, ErrUnexpectedEOL
	}
	switch input[pos] {
	case '0':
		return Bool(false), pos + 1, nil
	case '1':
		return Bool(true), pos + 1, nil
	}
	return nil, pos, ErrUnrecognized
}

func skipSpaces(input []byte, pos int) int {
	for pos < len(input) && input[pos] == ' ' {
		pos++
	}
	return pos
}

func isPrint(b byte) bool {
	return ' ' <= b && b <= '~'
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

func isAlpha(b byte) bool {
	return isLower(b) || isUpper(b)
}

func isUpper(b byte) bool {
	return 'A' <= b && b <= 'Z'
}

func isLower(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func isKeyChar(b byte) bool {
	if isLower(b) || isDigit(b) {
		return true
	}
	switch b {
	case '_', '-', '.', '*':
		return true
	}
	return false
}

func isTokenChar(b byte) bool {
	if isAlpha(b) || isDigit(b) {
		return true
	}
	switch b {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-',
		'.', '^', '_', '`', '|', '~', ':', '/':
		return true
	}
	return false
}

func isBase64Char(b byte) bool {
	if isAlpha(b) || isDigit(b) {
		return true
	}
	switch b {
	case '+', '/', '=':
		return true
	}
	return false
}
