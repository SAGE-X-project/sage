package jcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sample from RFC 8785 Section 3.2.3 / Appendix B (input uses JSON escapes).
func TestCanonicalize_RFC8785Sample(t *testing.T) {
	in := `{
  "numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001],
  "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/",
  "literals": [null, true, false]
}`
	want := "{\"literals\":[null,true,false],\"numbers\":[333333333.3333333,1e+30,4.5,0.002,1e-27],\"string\":\"\u20ac$\\u000f\\nA'B\\\"\\\\\\\\\\\"/\"}"
	out, err := Canonicalize([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, want, string(out))
}

// Key ordering sample from RFC 8785 Section 3.2.3.
func TestCanonicalize_KeyOrderIsUTF16(t *testing.T) {
	in := `{"\u20ac":"Euro Sign","\r":"Carriage Return","\ufb33":"Hebrew Letter Dalet With Dagesh","1":"One","\ud83d\ude00":"Emoji: Grinning Face","\u0080":"Control","\u00f6":"Latin Small Letter O With Diaeresis"}`
	want := "{\"\\r\":\"Carriage Return\",\"1\":\"One\",\"\u0080\":\"Control\",\"\u00f6\":\"Latin Small Letter O With Diaeresis\",\"\u20ac\":\"Euro Sign\",\"\U0001F600\":\"Emoji: Grinning Face\",\"\ufb33\":\"Hebrew Letter Dalet With Dagesh\"}"
	out, err := Canonicalize([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, want, string(out))
}

func TestCanonicalize_NumberFormats(t *testing.T) {
	cases := map[string]string{
		`0`: "0", `-0`: "0", `1`: "1", `-1.5`: "-1.5", `100`: "100", `1e21`: "1e+21", `1e20`: "100000000000000000000",
		`0.000001`: "0.000001", `0.0000001`: "1e-7", `123456789012345680000`: "123456789012345680000",
		`5e-324`: "5e-324", `1.7976931348623157e308`: "1.7976931348623157e+308", `9007199254740993`: "9007199254740992",
	}
	for in, want := range cases {
		out, err := Canonicalize([]byte(in))
		require.NoError(t, err, in)
		assert.Equal(t, want, string(out), in)
	}
}

func TestMarshal_NoHTMLEscapingAndStableOutput(t *testing.T) {
	v := map[string]interface{}{"b": "<a&b>", "a": []interface{}{1.0, "x"}, "c": map[string]interface{}{"z": nil, "y": true}}
	out, err := Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, `{"a":[1,"x"],"b":"<a&b>","c":{"y":true,"z":null}}`, string(out))
	again, err := Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, out, again)
}

func TestCanonicalize_Rejects(t *testing.T) {
	_, err := Canonicalize([]byte(`{"a":1} x`))
	require.Error(t, err)
	_, err = Canonicalize([]byte(`{"a":`))
	require.Error(t, err)
	_, err = Canonicalize([]byte("\"\xff\""))
	require.Error(t, err)
}
