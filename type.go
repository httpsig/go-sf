package sf

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Encoder defines a structured-field type with support for encoding.
type Encoder interface {

	// Encode serializes a structured field.
	Encode() string
}

// List is an array of zero or more list members.
type List []Member

// Encode serializes the list.
func (l List) Encode() string {
	if len(l) == 0 {
		return ""
	}
	members := make([]string, 0, len(l))
	for _, m := range l {
		members = append(members, m.Encode())
	}
	return strings.Join(members, ", ")
}

// Dict is an ordered map of key-value pairs.
type Dict []Pair

// Encode serializes the dictionary.
func (d Dict) Encode() string {
	if len(d) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(d))
	for _, p := range d {
		pairs = append(pairs, p.Encode())
	}
	return strings.Join(pairs, ", ")
}

// Add adds a new pair to the dictionary and returns the modified dictionary.
func (d Dict) Add(k string, v Member) Dict {
	for _, p := range d {
		if p.Key == k {
			p.Value = v
			return d
		}
	}
	d = append(d, Pair{k, v})
	return d
}

// Get retrieves a value from the dictionary by its key.
func (d Dict) Get(k string) Member {
	for _, p := range d {
		if p.Key == k {
			return p.Value
		}
	}
	return nil
}

// Pair is a key-value pair in a dictionary.
type Pair struct {
	Key   string
	Value Member
}

// Encode serializes the key-value pair.
func (p *Pair) Encode() string {
	if it, isItem := p.Value.(*Item); isItem {
		if v, isBool := it.Bare.(Bool); isBool && bool(v) {
			return p.Key + it.Params.Encode()
		}
	}
	return p.Key + "=" + p.Value.Encode()
}

// Member defines a list member item or dictionary member value, i.e. an item or
// an inner list.
type Member interface {
	Encoder
	isMember()
}

// InnerList is an array of zero or more items having zero or more associated
// parameters.
type InnerList struct {
	Items  []Item
	Params ParamList
}

// Encode serializes the inner list.
func (i *InnerList) Encode() string {
	if len(i.Items) == 0 {
		return "()" + i.Params.Encode()
	}
	items := make([]string, 0, len(i.Items))
	for _, it := range i.Items {
		items = append(items, it.Encode())
	}
	return "(" + strings.Join(items, " ") + ")" + i.Params.Encode()
}

// Item is a bare item having zero or more associated parameters.
type Item struct {
	Bare   BareItem
	Params ParamList
}

// Encode serializes the item.
func (i *Item) Encode() string {
	return i.Bare.Encode() + i.Params.Encode()
}

// ParamList is an array of zero or more parameters.
type ParamList []Param

// Add adds a new parameter to the list and returns the modified list.
func (l ParamList) Add(k string, v BareItem) ParamList {
	for _, p := range l {
		if p.Key == k {
			p.Value = v
			return l
		}
	}
	l = append(l, Param{k, v})
	return l
}

// Get retrieves a parameter value from the list by its key.
func (l ParamList) Get(k string) BareItem {
	for _, p := range l {
		if p.Key == k {
			return p.Value
		}
	}
	return nil
}

// Encode serializes the parameter list.
func (l ParamList) Encode() string {
	if len(l) == 0 {
		return ""
	}
	params := make([]string, 0, len(l))
	for _, p := range l {
		params = append(params, p.Encode())
	}
	return ";" + strings.Join(params, ";")
}

// Param is a key-value pair that is associated with an item.
type Param struct {
	Key   string
	Value BareItem
}

// Encode serializes the parameter.
func (p *Param) Encode() string {
	if v, isBool := p.Value.(Bool); isBool && bool(v) {
		return p.Key
	}
	return p.Key + "=" + p.Value.Encode()
}

// BareItem is an integer, a decimal, a string, a token, a byte sequence, or a
// boolean.
type BareItem interface {
	Encoder
	isBareItem()
}

// Integer is an integer item.
//
// Incompatibility note: RFC8941, Section 3.3.1, describes integers as signed
// numbers up to 15 digits. Here, we are using 64-bit integers that support much
// larger values. It is recommended to use strings or byte sequences for very
// large integers, since they may not be handled correctly buy other
// implementations.
type Integer int64

// Encode serializes the integer item.
func (i Integer) Encode() string {
	return strconv.FormatInt(int64(i), 10)
}

// Decimal is a number item with fractional component.
//
// Note: RFC8941, Section 3.3.2, accepts up to 3 digits for fractional
// component. Extra fractional digits will be rounded up or down.
//
// Incompatibility note: RFC8941, Section 3.3.2, describes decimals as signed
// numbers up to 12 digits. Here, we are using 64-bit integers that support much
// larger values. It is recommended to use strings or byte sequences for very
// large decimals, since they may not be handled correctly by other
// implementations.
type Decimal int64

// Encode serializes the decimal item.
func (d Decimal) Encode() string {
	var sign string
	if d < 0 {
		sign = "-"
		d = -d
	}
	intPart := d / 1000
	fracPart := d % 1000
	switch {
	case fracPart == 0:
		return fmt.Sprintf("%s%d.0", sign, intPart)
	case fracPart < 10:
		return remTrailZeros(fmt.Sprintf("%s%d.00%d", sign, intPart, fracPart))
	case fracPart < 100:
		return remTrailZeros(fmt.Sprintf("%s%d.0%d", sign, intPart, fracPart))
	default:
		return remTrailZeros(fmt.Sprintf("%s%d.%d", sign, intPart, fracPart))
	}
}

func remTrailZeros(d string) string {
	if last := len(d) - 1; d[last] == '0' {
		if nextToLast := len(d) - 2; d[nextToLast] == '0' {
			return d[:nextToLast]
		}
		return d[:last]
	}
	return d
}

// String is an ASCII string item.
type String string

// Encode serializes the string item.
func (s String) Encode() string {
	return strconv.QuoteToASCII(string(s))
}

// Token is a short textual word item.
type Token string

// Encode serializes the token item.
func (s Token) Encode() string {
	return string(s)
}

// ByteSeq is a Base-64 encoded byte sequence item.
type ByteSeq []byte

// Encode serializes the byte sequence item.
func (bs ByteSeq) Encode() string {
	return ":" + base64.StdEncoding.EncodeToString(bs) + ":"
}

// Bool is a boolean item.
type Bool bool

// Encode serializes the boolean item.
func (b Bool) Encode() string {
	if b {
		return "?1"
	}
	return "?0"
}

func (i InnerList) isMember() {}
func (i Item) isMember()      {}

func (Integer) isBareItem() {}
func (Decimal) isBareItem() {}
func (String) isBareItem()  {}
func (Token) isBareItem()   {}
func (ByteSeq) isBareItem() {}
func (Bool) isBareItem()    {}
