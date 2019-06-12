// Copyright [2019] LinkedIn Corp. Licensed under the Apache License, Version
// 2.0 (the "License"); you may not use this file except in compliance with the
// License.  You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

package goavro

import (
	"bytes"
	"fmt"
	"testing"
)

func ExampleCodecCanonicalSchema() {
	schema := `{"type":"map","values":{"type":"enum","name":"foo","symbols":["alpha","bravo"]}}`
	codec, err := NewCodec(schema)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(codec.CanonicalSchema())
	}
	// Output: {"type":"map","values":{"name":"foo","type":"enum","symbols":["alpha","bravo"]}}
}

func TestCodecSchemaCRC64Avro(t *testing.T) {
	cases := []struct {
		Schema      string
		Fingerprint int64
	}{
		{
			Schema:      `"null"`,
			Fingerprint: 7195948357588979594,
		},
		{
			Schema:      `"boolean"`,
			Fingerprint: -6970731678124411036,
		},
		{
			Schema:      `"int"`,
			Fingerprint: 8247732601305521295,
		},
		{
			Schema:      `"long"`,
			Fingerprint: -3434872931120570953,
		},
		{
			Schema:      `"float"`,
			Fingerprint: 5583340709985441680,
		},
		{
			Schema:      `"double"`,
			Fingerprint: -8181574048448539266,
		},
		{
			Schema:      `"bytes"`,
			Fingerprint: 5746618253357095269,
		},
		{
			Schema:      `"string"`,
			Fingerprint: -8142146995180207161,
		},
		{
			Schema:      `[ "int"  ]`,
			Fingerprint: -5232228896498058493,
		},
		{
			Schema:      `[ "int" , {"type":"boolean"} ]`,
			Fingerprint: 5392556393470105090,
		},
		{
			Schema:      `{"fields":[], "type":"record", "name":"foo"}`,
			Fingerprint: -4824392279771201922,
		},
		{
			Schema:      `{"fields":[], "type":"record", "name":"foo", "namespace":"x.y"}`,
			Fingerprint: 5916914534497305771,
		},
		{
			Schema:      `{"fields":[], "type":"record", "name":"a.b.foo", "namespace":"x.y"}`,
			Fingerprint: -4616218487480524110,
		},
		{
			Schema:      `{"fields":[], "type":"record", "name":"foo", "doc":"Useful info"}`,
			Fingerprint: -4824392279771201922,
		},
		{
			Schema:      `{"fields":[], "type":"record", "name":"foo", "aliases":["foo","bar"]}`,
			Fingerprint: -4824392279771201922,
		},
		{
			Schema:      `{"fields":[], "type":"record", "name":"foo", "doc":"foo", "aliases":["foo","bar"]}`,
			Fingerprint: -4824392279771201922,
		},
		{
			Schema:      `{"fields":[{"type":{"type":"boolean"}, "name":"f1"}], "type":"record", "name":"foo"}`,
			Fingerprint: 7843277075252814651,
		},
		{
			Schema:      `{ "fields":[{"type":"boolean", "aliases":[], "name":"f1", "default":true}, {"order":"descending","name":"f2","doc":"Hello","type":"int"}], "type":"record", "name":"foo"}`,
			Fingerprint: -4860222112080293046,
		},
		{
			Schema:      `{"type":"enum", "name":"foo", "symbols":["A1"]}`,
			Fingerprint: -6342190197741309591,
		},
		{
			Schema:      `{"namespace":"x.y.z", "type":"enum", "name":"foo", "doc":"foo bar", "symbols":["A1", "A2"]}`,
			Fingerprint: -4448647247586288245,
		},
		{
			Schema:      `{"name":"foo","type":"fixed","size":15}`,
			Fingerprint: 1756455273707447556,
		},
		{
			Schema:      `{"namespace":"x.y.z", "type":"fixed", "name":"foo", "doc":"foo bar", "size":32}`,
			Fingerprint: -3064184465700546786,
		},
		{
			Schema:      `{ "items":{"type":"null"}, "type":"array"}`,
			Fingerprint: -589620603366471059,
		},
		{
			Schema:      `{ "values":"string", "type":"map"}`,
			Fingerprint: -8732877298790414990,
		},
		{
			Schema:      `{"name":"PigValue","type":"record", "fields":[{"name":"value", "type":["null", "int", "long", "PigValue"]}]}`,
			Fingerprint: -1759257747318642341,
		},
	}

	for _, c := range cases {
		codec, err := NewCodec(c.Schema)
		if err != nil {
			t.Fatalf("CASE: %s; cannot create code: %s", c.Schema, err)
		}
		if got, want := codec.SchemaCRC64Avro(), c.Fingerprint; got != want {
			t.Errorf("CASE: %s; GOT: %#x; WANT: %#x", c.Schema, got, want)
		}
	}
}

func TestSingleObjectEncoding(t *testing.T) {
	t.Run("int", func(*testing.T) {
		schema := `"int"`

		codec, err := NewCodec(schema)
		if err != nil {
			t.Fatalf("cannot create code: %s", err)
		}

		t.Run("encoding", func(t *testing.T) {
			t.Run("does not modify source buf when cannot encode", func(t *testing.T) {
				buf := []byte{0xDE, 0xAD, 0xBE, 0xEF}

				buf, err = codec.singleFromNative(buf, "strings cannot be encoded as int")
				ensureError(t, err, "cannot encode binary int")

				if got, want := buf, []byte("\xDE\xAD\xBE\xEF"); !bytes.Equal(got, want) {
					t.Errorf("GOT: %v; WANT: %v", got, want)
				}
			})

			t.Run("appends header then encoded data", func(t *testing.T) {
				const original = "\x01\x02\x03\x04"
				buf := []byte(original)

				buf, err = codec.singleFromNative(buf, 3)
				ensureError(t, err)

				fp := "\xC3\x01" + "\x8F\x5C\x39\x3F\x1A\xD5\x75\x72"

				if got, want := buf, []byte(original+fp+"\x06"); !bytes.Equal(got, want) {
					t.Errorf("\nGOT:\n\t%v;\nWANT:\n\t%v", got, want)
				}
			})
		})

		t.Run("decoding", func(t *testing.T) {
			const original = ""
			buf := []byte(original)

			buf, err = codec.singleFromNative(nil, 3)
			ensureError(t, err)

			buf = append(buf, "\xDE\xAD"...) // append some junk

			datum, newBuf, err := codec.nativeFromSingle(buf)
			ensureError(t, err)

			if got, want := datum, int32(3); got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}

			// ensure junk is left alone
			if got, want := newBuf, []byte("\xDE\xAD"); !bytes.Equal(got, want) {
				t.Errorf("\nGOT:\n\t%q;\nWANT:\n\t%q", got, want)
			}
		})
	})

	t.Run("record round trip", func(t *testing.T) {
		codec, err := NewCodec(`
{
  "type": "record",
  "name": "LongList",                  
  "fields" : [
    {"name": "next", "type": ["null", "LongList"], "default": null}
  ]
}
`)
		ensureError(t, err)

		// NOTE: May omit fields when using default value
		initial := `{"next":{"LongList":{}}}`

		// NOTE: Textual encoding will show all fields, even those with values that
		// match their default values
		final := `{"next":{"LongList":{"next":null}}}`

		// Convert textual Avro data (in Avro JSON format) to native Go form
		datum, _, err := codec.NativeFromTextual([]byte(initial))
		ensureError(t, err)

		// Convert native Go form to single-object encoding form
		buf, err := codec.singleFromNative(nil, datum)
		ensureError(t, err)

		// Convert single-object encoding form back to native Go form
		datum, _, err = codec.nativeFromSingle(buf)
		ensureError(t, err)

		// Convert native Go form to textual Avro data
		buf, err = codec.TextualFromNative(nil, datum)
		ensureError(t, err)

		if got, want := string(buf), final; got != want {
			t.Fatalf("GOT: %v; WANT: %v", got, want)
		}
	})
}
