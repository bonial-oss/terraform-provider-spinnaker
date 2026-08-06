package api

import (
	"net/http"

	"github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api/errors"
	"github.com/antihax/optional"
	"github.com/mitchellh/mapstructure"
	gateapi "github.com/spinnaker/spin/gateapi"
)

// CreatePipelineTemplateV2 creates a pipeline template.
func CreatePipelineTemplateV2(client *Client, template *PipelineTemplateV2) error {
	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		return client.V2PipelineTemplatesControllerApi.CreateUsingPOST1(client.Context, template, nil)
	})
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated) {
		return errors.NewResponseError(resp, err)
	}

	return nil
}

// GetPipelineTemplateV2 fetches the pipeline template with templateID.
func GetPipelineTemplateV2(client *Client, templateID string) (*PipelineTemplateV2, error) {
	payload, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		return client.V2PipelineTemplatesControllerApi.GetUsingGET2(client.Context, templateID, nil)
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.NewResponseError(resp, err)
	}

	var template PipelineTemplateV2

	if err := mapstructure.Decode(payload, &template); err != nil {
		return nil, err
	}

	return &template, nil
}

// DeletePipelineTemplateV2 deletes the pipeline template with templateID.
// Either digest or tag can be set on a delete request, but not both.
func DeletePipelineTemplateV2(client *Client, templateID, tag, digest string) error {
	opts := &gateapi.V2PipelineTemplatesControllerApiDeleteUsingDELETE1Opts{}
	if digest != "" {
		opts.Digest = optional.NewString(digest)
	} else if tag != "" {
		opts.Tag = optional.NewString(tag)
	}

	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		return client.V2PipelineTemplatesControllerApi.DeleteUsingDELETE1(client.Context, templateID, opts)
	})
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent) {
		return errors.NewResponseError(resp, err)
	}

	return nil
}

// UpdatePipelineTemplateV2 updates the pipeline template with templateID with
// the data in template.
func UpdatePipelineTemplateV2(client *Client, template *PipelineTemplateV2) error {
	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		return client.V2PipelineTemplatesControllerApi.UpdateUsingPOST1(client.Context, template.ID, template, nil)
	})
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated) {
		return errors.NewResponseError(resp, err)
	}

	return nil
}

// ListPipelineTemplateV2Versions lists versions of all available pipeline
// templates. The resulting map is keyed by template ID.
func ListPipelineTemplateV2Versions(client *Client) (map[string][]*PipelineTemplateV2Version, error) {
	var payload interface{}

	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		v, resp, err := client.V2PipelineTemplatesControllerApi.ListVersionsUsingGET(client.Context, nil)
		payload = v
		return nil, resp, err
	})
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.NewResponseError(resp, err)
	}

	var versionMap map[string][]*PipelineTemplateV2Version

	if err = mapstructure.Decode(payload, &versionMap); err != nil {
		return nil, err
	}

	return versionMap, nil
}
