package fuego

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

type MyStruct struct {
	B string `json:"b"`
	C int    `json:"c"`
	D bool   `json:"d"`
}

type MyOutputStruct struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type testCaseForTagType[V any] struct {
	name        string
	description string
	inputType   V

	expectedTagValue     string
	expectedTagValueType *openapi3.Types
}

func Test_tagFromType(t *testing.T) {
	s := NewServer()
	type DeeplyNested *[]MyStruct
	type MoreDeeplyNested *[]DeeplyNested

	tcs := []testCaseForTagType[any]{
		{
			name:        "unknown_interface",
			description: "behind any interface",
			inputType:   *new(any),

			expectedTagValue: "unknown-interface",
		},
		{
			name:        "simple_struct",
			description: "basic struct",
			inputType:   MyStruct{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"object"},
		},
		{
			name:        "is_pointer",
			description: "",
			inputType:   &MyStruct{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"object"},
		},
		{
			name:        "is_array",
			description: "",
			inputType:   []MyStruct{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "is_reference_to_array",
			description: "",
			inputType:   &[]MyStruct{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "is_deeply_nested",
			description: "behind 4 pointers",
			inputType:   new(DeeplyNested),

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "5_pointers",
			description: "behind 5 pointers",
			inputType:   *new(MoreDeeplyNested),

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "6_pointers",
			description: "behind 6 pointers",
			inputType:   new(MoreDeeplyNested),

			expectedTagValue:     "default",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "7_pointers",
			description: "behind 7 pointers",
			inputType:   []*MoreDeeplyNested{},

			expectedTagValue: "default",
		},
		{
			name:        "detecting_string",
			description: "",
			inputType:   "string",

			expectedTagValue:     "string",
			expectedTagValueType: &openapi3.Types{"string"},
		},
		{
			name:        "new_string",
			description: "",
			inputType:   new(string),

			expectedTagValue:     "string",
			expectedTagValueType: &openapi3.Types{"string"},
		},
		{
			name:        "string_array",
			description: "",
			inputType:   []string{},

			expectedTagValue:     "string",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "pointer_string_array",
			description: "",
			inputType:   &[]string{},

			expectedTagValue:     "string",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "DataOrTemplate",
			description: "",
			inputType:   DataOrTemplate[MyStruct]{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"object"},
		},
		{
			name:        "ptr to DataOrTemplate",
			description: "",
			inputType:   &DataOrTemplate[MyStruct]{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"object"},
		},
		{
			name:        "DataOrTemplate of an array",
			description: "",
			inputType:   DataOrTemplate[[]MyStruct]{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "ptr to DataOrTemplate of an array of ptr",
			description: "",
			inputType:   &DataOrTemplate[[]*MyStruct]{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "ptr to DataOrTemplate of a ptr to an array",
			description: "",
			inputType:   &DataOrTemplate[*[]MyStruct]{},

			expectedTagValue:     "MyStruct",
			expectedTagValueType: &openapi3.Types{"array"},
		},
		{
			name:        "ptr to DataOrTemplate of a ptr to an array of ptr",
			description: "",
			inputType:   &DataOrTemplate[*[]*MyStruct]{},

			expectedTagValue:     "default",
			expectedTagValueType: &openapi3.Types{"array"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tag := schemaTagFromType(s, tc.inputType)
			require.Equal(t, tc.expectedTagValue, tag.name, tc.description)
			if tc.expectedTagValueType != nil {
				require.NotNil(t, tag.Value)
				require.Equal(t, tc.expectedTagValueType, tag.Value.Type, tc.description)
			}
		})
	}
}

func TestServer_generateOpenAPI(t *testing.T) {
	s := NewServer()
	Get(s, "/", func(*ContextNoBody) (MyStruct, error) {
		return MyStruct{}, nil
	})
	Post(s, "/post", func(*ContextWithBody[MyStruct]) ([]MyStruct, error) {
		return nil, nil
	})
	Get(s, "/post/{id}", func(*ContextNoBody) (MyOutputStruct, error) {
		return MyOutputStruct{}, nil
	})
	document := s.OutputOpenAPISpec()
	require.NotNil(t, document)
	require.NotNil(t, document.Paths.Find("/"))
	require.Nil(t, document.Paths.Find("/unknown"))
	require.NotNil(t, document.Paths.Find("/post"))
	require.Equal(t, document.Paths.Find("/post").Post.Responses.Value("200").Value.Content["application/json"].Schema.Value.Type, &openapi3.Types{"array"})
	require.Equal(t, document.Paths.Find("/post").Post.Responses.Value("200").Value.Content["application/json"].Schema.Value.Items.Ref, "#/components/schemas/MyStruct")
	require.NotNil(t, document.Paths.Find("/post/{id}").Get.Responses.Value("200"))
	require.NotNil(t, document.Paths.Find("/post/{id}").Get.Responses.Value("200").Value.Content["application/json"])
	require.Nil(t, document.Paths.Find("/post/{id}").Get.Responses.Value("200").Value.Content["application/json"].Schema.Value.Properties["unknown"])
	require.Equal(t, document.Paths.Find("/post/{id}").Get.Responses.Value("200").Value.Content["application/json"].Schema.Value.Properties["quantity"].Value.Type, &openapi3.Types{"integer"})

	t.Run("openapi doc is available through a route", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/swagger/openapi.json", nil)
		s.Mux.ServeHTTP(w, r)

		require.Equal(t, 200, w.Code)
	})
}

func TestServer_OutputOpenApiSpec(t *testing.T) {
	docPath := "doc/openapi.json"
	t.Run("base", func(t *testing.T) {
		s := NewServer(
			WithOpenAPIConfig(
				OpenAPIConfig{
					JsonFilePath: docPath,
				},
			),
		)
		Get(s, "/", func(*ContextNoBody) (MyStruct, error) {
			return MyStruct{}, nil
		})

		document := s.OutputOpenAPISpec()
		require.NotNil(t, document)

		file, err := os.Open(docPath)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer os.Remove(file.Name())
		require.Equal(t, 1, lineCounter(t, file))
	})
	t.Run("do not print file", func(t *testing.T) {
		s := NewServer(
			WithOpenAPIConfig(
				OpenAPIConfig{
					JsonFilePath:     docPath,
					DisableLocalSave: true,
				},
			),
		)
		Get(s, "/", func(*ContextNoBody) (MyStruct, error) {
			return MyStruct{}, nil
		})

		document := s.OutputOpenAPISpec()
		require.NotNil(t, document)

		file, err := os.Open(docPath)
		require.Error(t, err)
		require.Nil(t, file)
	})
	t.Run("swagger disabled", func(t *testing.T) {
		s := NewServer(
			WithOpenAPIConfig(
				OpenAPIConfig{
					JsonFilePath:     docPath,
					DisableLocalSave: true,
					DisableSwagger:   true,
				},
			),
		)
		Get(s, "/", func(*ContextNoBody) (MyStruct, error) {
			return MyStruct{}, nil
		})

		document := s.OutputOpenAPISpec()
		require.Len(t, document.Paths.Map(), 1)
		require.NotNil(t, document)

		file, err := os.Open(docPath)
		require.Error(t, err)
		require.Nil(t, file)
	})
	t.Run("pretty format json file", func(t *testing.T) {
		s := NewServer(
			WithOpenAPIConfig(
				OpenAPIConfig{
					JsonFilePath:     docPath,
					PrettyFormatJson: true,
				},
			),
		)
		Get(s, "/", func(*ContextNoBody) (MyStruct, error) {
			return MyStruct{}, nil
		})

		document := s.OutputOpenAPISpec()
		require.NotNil(t, document)

		file, err := os.Open(docPath)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer os.Remove(file.Name())
		require.Greater(t, lineCounter(t, file), 1)
	})
}

func lineCounter(t *testing.T, r io.Reader) int {
	buf := make([]byte, 32*1024)
	count := 1
	lineSep := []byte{'\n'}

	c, err := r.Read(buf)
	require.NoError(t, err)
	count += bytes.Count(buf[:c], lineSep)
	return count
}

func BenchmarkRoutesRegistration(b *testing.B) {
	for range b.N {
		s := NewServer(
			WithoutLogger(),
		)
		Get(s, "/", func(ContextNoBody) (MyStruct, error) {
			return MyStruct{}, nil
		})
		for j := 0; j < 100; j++ {
			Post(s, fmt.Sprintf("/post/%d", j), func(*ContextWithBody[MyStruct]) ([]MyStruct, error) {
				return nil, nil
			})
		}
		for j := 0; j < 100; j++ {
			Get(s, fmt.Sprintf("/post/{id}/%d", j), func(ContextNoBody) (MyStruct, error) {
				return MyStruct{}, nil
			})
		}
	}
}

func BenchmarkServer_generateOpenAPI(b *testing.B) {
	for range b.N {
		s := NewServer(
			WithoutLogger(),
		)
		Get(s, "/", func(ContextNoBody) (MyStruct, error) {
			return MyStruct{}, nil
		})
		for j := 0; j < 100; j++ {
			Post(s, fmt.Sprintf("/post/%d", j), func(*ContextWithBody[MyStruct]) ([]MyStruct, error) {
				return nil, nil
			})
		}
		for j := 0; j < 100; j++ {
			Get(s, fmt.Sprintf("/post/{id}/%d", j), func(ContextNoBody) (MyStruct, error) {
				return MyStruct{}, nil
			})
		}

		s.OutputOpenAPISpec()
	}
}

func TestValidateJsonSpecLocalPath(t *testing.T) {
	require.Equal(t, true, validateJsonSpecLocalPath("path/to/jsonSpec.json"))
	require.Equal(t, true, validateJsonSpecLocalPath("spec.json"))
	require.Equal(t, true, validateJsonSpecLocalPath("path_/jsonSpec.json"))
	require.Equal(t, true, validateJsonSpecLocalPath("Path_2000-12-08/json_Spec-007.json"))
	require.Equal(t, false, validateJsonSpecLocalPath("path/to/jsonSpec"))
	require.Equal(t, false, validateJsonSpecLocalPath("path/to/jsonSpec.jsn"))
	require.Equal(t, false, validateJsonSpecLocalPath("path.to/js?.test.jsn"))
}

func TestValidateJsonSpecUrl(t *testing.T) {
	require.Equal(t, true, validateJsonSpecUrl("/path/to/jsonSpec.json"))
	require.Equal(t, true, validateJsonSpecUrl("/spec.json"))
	require.Equal(t, true, validateJsonSpecUrl("/path_/jsonSpec.json"))
	require.Equal(t, false, validateJsonSpecUrl("path/to/jsonSpec.json"))
	require.Equal(t, false, validateJsonSpecUrl("/path/to/jsonSpec"))
	require.Equal(t, false, validateJsonSpecUrl("/path/to/jsonSpec.jsn"))
}

func TestValidateSwaggerUrl(t *testing.T) {
	require.Equal(t, true, validateSwaggerUrl("/path/to/jsonSpec"))
	require.Equal(t, true, validateSwaggerUrl("/swagger"))
	require.Equal(t, true, validateSwaggerUrl("/Super-usefull_swagger-2000"))
	require.Equal(t, true, validateSwaggerUrl("/Super-usefull_swagger-"))
	require.Equal(t, true, validateSwaggerUrl("/Super-usefull_swagger__"))
	require.Equal(t, true, validateSwaggerUrl("/Super-usefull_swaggeR"))
	require.Equal(t, false, validateSwaggerUrl("/spec.json"))
	require.Equal(t, false, validateSwaggerUrl("/path_/swagger.json"))
	require.Equal(t, false, validateSwaggerUrl("path/to/jsonSpec."))
	require.Equal(t, false, validateSwaggerUrl("path/to/jsonSpec%"))
}

func TestLocalSave(t *testing.T) {
	s := NewServer()
	t.Run("with valid path", func(t *testing.T) {
		err := s.saveOpenAPIToFile("/tmp/jsonSpec.json", []byte("test"))
		require.NoError(t, err)

		// cleanup
		os.Remove("/tmp/jsonSpec.json")
	})

	t.Run("with invalid path", func(t *testing.T) {
		err := s.saveOpenAPIToFile("///jsonSpec.json", []byte("test"))
		require.Error(t, err)
	})
}

func TestAutoGroupTags(t *testing.T) {
	s := NewServer(
		WithOpenAPIConfig(OpenAPIConfig{
			DisableLocalSave: true,
			DisableSwagger:   true,
		}),
	)
	Get(s, "/a", func(*ContextNoBody) (MyStruct, error) {
		return MyStruct{}, nil
	})

	group := Group(s, "/group")
	Get(group, "/b", func(*ContextNoBody) (MyStruct, error) {
		return MyStruct{}, nil
	})

	subGroup := Group(group, "/subgroup")
	Get(subGroup, "/c", func(*ContextNoBody) (MyStruct, error) {
		return MyStruct{}, nil
	})

	otherGroup := Group(s, "/other")
	Get(otherGroup, "/d", func(*ContextNoBody) (MyStruct, error) {
		return MyStruct{}, nil
	})

	document := s.OutputOpenAPISpec()
	require.NotNil(t, document)
	require.Nil(t, document.Paths.Find("/a").Get.Tags)
	require.Equal(t, []string{"group"}, document.Paths.Find("/group/b").Get.Tags)
	require.Equal(t, []string{"subgroup"}, document.Paths.Find("/group/subgroup/c").Get.Tags)
	require.Equal(t, []string{"other"}, document.Paths.Find("/other/d").Get.Tags)
}
