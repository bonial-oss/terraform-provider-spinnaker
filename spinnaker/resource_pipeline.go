package spinnaker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api"
	apierrors "github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api/errors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePipeline() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"application": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"pipeline": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: pipelineDiffSuppressFunc,
			},
			"pipeline_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		Create: resourcePipelineCreate,
		Read:   resourcePipelineRead,
		Update: resourcePipelineUpdate,
		Delete: resourcePipelineDelete,
		Exists: resourcePipelineExists,
	}
}

type pipelineRead struct {
	Name        string `json:"name"`
	Application string `json:"application"`
	ID          string `json:"id"`
}

func resourcePipelineCreate(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	applicationName := data.Get("application").(string)
	pipelineName := data.Get("name").(string)
	rawPipeline := data.Get("pipeline").(string)

	pipeline, err := parsePipeline(rawPipeline)
	if err != nil {
		return err
	}

	pipeline["application"] = applicationName
	pipeline["name"] = pipelineName
	delete(pipeline, "id")

	err = api.CreatePipeline(client, pipeline)
	if apierrors.IsPipelineAlreadyExists(err) {
		err = api.RecreatePipeline(client, applicationName, pipelineName, pipeline)
	}

	if err != nil {
		return fmt.Errorf("failed to create pipeline %q for application %q: %w",
			pipelineName, applicationName, err)
	}

	return resourcePipelineRead(data, meta)
}

func resourcePipelineRead(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	applicationName := data.Get("application").(string)
	pipelineName := data.Get("name").(string)

	var p pipelineRead

	pipeline, err := api.GetPipeline(client, applicationName, pipelineName, &p)
	if apierrors.IsNotFound(err) {
		data.SetId("")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to fetch pipeline %q for application %q: %w",
			pipelineName, applicationName, err)
	}

	encodedPipeline, err := editAndEncodePipeline(pipeline)
	if err != nil {
		return err
	}

	if err := data.Set("pipeline", encodedPipeline); err != nil {
		return err
	}

	if err := data.Set("pipeline_id", p.ID); err != nil {
		return err
	}

	data.SetId(p.ID)

	return nil
}

func resourcePipelineUpdate(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	applicationName := data.Get("application").(string)
	pipelineName := data.Get("name").(string)
	pipelineID := data.Get("pipeline_id").(string)
	rawPipeline := data.Get("pipeline").(string)

	pipeline, err := parsePipeline(rawPipeline)
	if err != nil {
		return err
	}

	pipeline["application"] = applicationName
	pipeline["name"] = pipelineName
	pipeline["id"] = pipelineID

	err = api.UpdatePipeline(client, pipelineID, pipeline)
	if apierrors.IsPipelineAlreadyExists(err) {
		// Although it seems odd, this error can happen here due to the hideous
		// spinnaker API. We handle it by just recreating the pipeline.
		err = api.RecreatePipeline(client, applicationName, pipelineName, pipeline)
	}

	if err != nil {
		return fmt.Errorf("failed to update pipeline %q for application %q: %w",
			pipelineName, applicationName, err)
	}

	return resourcePipelineRead(data, meta)
}

func resourcePipelineDelete(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	applicationName := data.Get("application").(string)
	pipelineName := data.Get("name").(string)

	err = api.DeletePipeline(client, applicationName, pipelineName)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete pipeline %q for application %q: %w",
			pipelineName, applicationName, err)
	}

	return nil
}

func resourcePipelineExists(data *schema.ResourceData, meta interface{}) (bool, error) {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return false, err
	}

	applicationName := data.Get("application").(string)
	pipelineName := data.Get("name").(string)

	var p pipelineRead

	_, err = api.GetPipeline(client, applicationName, pipelineName, &p)
	if err != nil {
		// Weird error case: sometimes spinnaker returns an EOF error for non-existing pipelines.
		if apierrors.IsNotFound(err) || strings.Contains(err.Error(), "EOF") {
			return false, nil
		}

		return false, fmt.Errorf("failed to fetch pipeline %q for application %q: %w",
			pipelineName, applicationName, err)
	}

	return true, nil
}

func pipelineDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	// Spinnaker does non-trivial modifications to the JSON for a pipeline,
	// so we round-trip decode, edit, and encode the user's pipeline
	// spec, and compare against the decoded, edited, and encoded new pipeline.
	editedOld, err := decodeEditAndEncodePipeline(old)
	if err != nil {
		return false
	}

	editedNew, err := decodeEditAndEncodePipeline(new)
	if err != nil {
		return false
	}

	return editedOld == editedNew
}

func parsePipeline(rawPipeline string) (map[string]interface{}, error) {
	var pipeline map[string]interface{}

	if err := json.Unmarshal([]byte(rawPipeline), &pipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline json: %w", err)
	}

	return pipeline, nil
}

func decodeEditAndEncodePipeline(rawPipeline string) (string, error) {
	pipeline, err := parsePipeline(rawPipeline)
	if err != nil {
		return "", err
	}

	return editAndEncodePipeline(pipeline)
}

func editAndEncodePipeline(pipeline map[string]interface{}) (string, error) {
	// Remove the keys we know are problematic because they are managed
	// by spinnaker or are handled by other schema attributes.
	delete(pipeline, "application")
	delete(pipeline, "lastModifiedBy")
	delete(pipeline, "id")
	delete(pipeline, "index")
	delete(pipeline, "name")
	delete(pipeline, "updateTs")

	encoded, err := json.Marshal(pipeline)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	return string(encoded), nil
}
