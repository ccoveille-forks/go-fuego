package openapi3

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestToSchema(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		s := ToSchema("")
		require.Equal(t, "string", s.Type)
	})

	t.Run("alias to string", func(t *testing.T) {
		type S string
		s := ToSchema(S(""))
		require.Equal(t, "string", s.Type)
	})

	t.Run("struct with a field alias to string", func(t *testing.T) {
		type MyAlias string
		type S struct {
			A MyAlias
		}

		s := ToSchema(S{})
		require.Equal(t, "object", s.Type)
		require.Equal(t, "string", s.Properties["A"].Type)
	})

	t.Run("int", func(t *testing.T) {
		s := ToSchema(0)
		require.Equal(t, "integer", s.Type)
	})

	t.Run("bool", func(t *testing.T) {
		s := ToSchema(false)
		require.Equal(t, "boolean", s.Type)
	})

	t.Run("time", func(t *testing.T) {
		s := ToSchema(time.Now())
		require.Equal(t, "string", s.Type)
	})

	t.Run("struct", func(t *testing.T) {
		type S struct {
			A      string `json:"a" validate:"required" example:"hello"`
			B      int
			C      bool
			Nested struct {
				C string
			}
		}
		s := ToSchema(S{})
		require.Equal(t, "object", s.Type)
		require.Equal(t, "string", s.Properties["a"].Type)
		require.Equal(t, "integer", s.Properties["B"].Type)
		require.Equal(t, "boolean", s.Properties["C"].Type)
		require.Equal(t, []string{"a"}, s.Required)
		require.Equal(t, "object", s.Properties["Nested"].Type)
		require.Equal(t, "string", s.Properties["Nested"].Properties["C"].Type)

		gotSchema, err := json.Marshal(s)
		require.NoError(t, err)
		require.JSONEq(t, string(gotSchema), `
		{
			"type":"object",
			"required":["a"],
			"properties": {
				"a":{"type":"string","example":"hello"},
				"B":{"type":"integer"},
				"C":{"type":"boolean"},
				"Nested":{
					"type":"object",
					"properties":{
						"C":{"type":"string"}
					}
				}
			}
		}`)
	})

	t.Run("ptr to struct", func(t *testing.T) {
		type S struct {
			A      string
			B      int
			Nested struct {
				C string
			}
		}
		s := ToSchema(&S{})
		require.Equal(t, "object", s.Type)
		require.Equal(t, "string", s.Properties["A"].Type)
		require.Equal(t, "integer", s.Properties["B"].Type)
		// TODO require.Equal(t, []string{"A", "B", "Nested"}, s.Required)
		require.Equal(t, "object", s.Properties["Nested"].Type)
		require.Equal(t, "string", s.Properties["Nested"].Properties["C"].Type)

		gotSchema, err := json.Marshal(s)
		require.NoError(t, err)
		require.JSONEq(t, string(gotSchema), `
		{
			"type":"object",
			"properties": {
				"A":{"type":"string"},
				"B":{"type":"integer"},
				"Nested":{
					"type":"object",
					"properties":{
						"C":{"type":"string"}
					}
				}
			}
		}`)
	})

	t.Run("slice of strings", func(t *testing.T) {
		s := ToSchema([]string{})
		require.Equal(t, "array", s.Type)
		require.Equal(t, "string", s.Items.Type)
	})

	t.Run("slice of structs", func(t *testing.T) {
		type S struct {
			A string
		}
		s := ToSchema([]S{})
		require.Equal(t, "array", s.Type)
		require.Equal(t, "object", s.Items.Type)
		require.Equal(t, "string", s.Items.Properties["A"].Type)
	})

	t.Run("slice of ptr to struct", func(t *testing.T) {
		type S struct {
			A string
		}
		s := ToSchema([]*S{})
		require.Equal(t, "array", s.Type)
		require.Equal(t, "object", s.Items.Type)
		require.Equal(t, "string", s.Items.Properties["A"].Type)
	})

	t.Run("embedded struct", func(t *testing.T) {
		type S struct {
			A string
		}
		type T struct {
			S
			B int
		}
		s := ToSchema(T{})
		require.Equal(t, "object", s.Type)
		require.Equal(t, "", s.Properties["A"].Type)
		require.Equal(t, "object", s.Properties["S"].Type)
		require.Equal(t, "string", s.Properties["S"].Properties["A"].Type)
		require.Equal(t, "integer", s.Properties["B"].Type)
	})
}