package spinnaker

import (
	"testing"

	"github.com/bonial-oss/terraform-provider-spinnaker/spinnaker/api"
	"github.com/stretchr/testify/require"
)

func TestParsePipelineTemplateV2(t *testing.T) {
	t.Run("must be valid json", func(t *testing.T) {
		_, err := parsePipelineTemplateV2("{invalid")
		require.EqualError(t, err, `invalid template json: invalid character 'i' looking for beginning of object key string`)
	})

	t.Run("validates parsed template", func(t *testing.T) {
		_, err := parsePipelineTemplateV2(`{"variables":[{}]}`)
		require.Error(t, err)
		require.Regexp(t, `^invalid template: \d+ errors occurred:.*`, err.Error())
	})

	t.Run("parses valid template json", func(t *testing.T) {
		template, err := parsePipelineTemplateV2(`{"schema":"v2","pipeline":{},"metadata":{"name":"bar","description":"baz","scopes":["global"]}}`)
		require.NoError(t, err)

		expected := &api.PipelineTemplateV2{
			Schema: "v2",
			Metadata: api.PipelineTemplateV2Metadata{
				Name:        "bar",
				Description: "baz",
				Scopes:      []string{"global"},
			},
			Pipeline: map[string]interface{}{},
		}

		require.Equal(t, expected, template)
	})

	t.Run("validates that id field is not set", func(t *testing.T) {
		_, err := parsePipelineTemplateV2(`{"schema":"v2","id":"foo","pipeline":{},"metadata":{"name":"bar","description":"baz","scopes":["global"]}}`)
		require.Error(t, err)
		require.Regexp(t, `.*field 'id' must not be set.*`, err.Error())
	})
}
