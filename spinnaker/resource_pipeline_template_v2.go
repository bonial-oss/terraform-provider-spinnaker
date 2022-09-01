package spinnaker

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api"
	apierrors "github.com/Bonial-International-GmbH/terraform-provider-spinnaker/spinnaker/api/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePipelineTemplateV2() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a V2 pipeline template. See https://spinnaker.io/reference/pipeline/templates/ for more details.",
		Schema: map[string]*schema.Schema{
			"template": {
				Description: "JSON schema of the V2 pipeline template.",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					if _, err := parsePipelineTemplateV2(val.(string)); err != nil {
						errs = append(errs, fmt.Errorf("invalid pipeline template: %w", err))
					}
					return
				},
				DiffSuppressFunc: func(key, old, new string, d *schema.ResourceData) bool {
					equal, _ := areEqualJSON(old, new)
					return equal
				},
			},
			"template_id": {
				Description: "ID of the template.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"reference": {
				Description: "The URL for referencing the template in a pipeline instance.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
		Create: resourcePipelineTemplateV2Create,
		Read:   resourcePipelineTemplateV2Read,
		Update: resourcePipelineTemplateV2Update,
		Delete: resourcePipelineTemplateV2Delete,
		Exists: resourcePipelineTemplateV2Exists,
	}
}

func resourcePipelineTemplateV2Create(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	rawTemplate := data.Get("template").(string)

	template, err := parsePipelineTemplateV2(rawTemplate)
	if err != nil {
		return err
	}

	template.ID = data.Get("template_id").(string)

	if err := api.CreatePipelineTemplateV2(client, template); err != nil {
		return fmt.Errorf("failed to create pipeline template: %w", err)
	}

	data.SetId(template.ID)

	return resourcePipelineTemplateV2Read(data, meta)
}

func resourcePipelineTemplateV2Read(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	templateID := data.Get("template_id").(string)

	template, err := api.GetPipelineTemplateV2(client, templateID)
	if apierrors.IsNotFound(err) {
		data.SetId("")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to fetch pipeline template %q: %w", templateID, err)
	}

	// Unset template ID before marshalling so it does not cause a diff in the
	// template field.
	template.ID = ""

	rawTemplate, err := json.Marshal(template)
	if err != nil {
		return err
	}

	reference := fmt.Sprintf("spinnaker://%s", templateID)

	if err := data.Set("template", string(rawTemplate)); err != nil {
		return err
	}

	if err := data.Set("template_id", templateID); err != nil {
		return err
	}

	if err := data.Set("reference", reference); err != nil {
		return err
	}

	data.SetId(templateID)

	return nil
}

func resourcePipelineTemplateV2Update(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	rawTemplate := data.Get("template").(string)

	template, err := parsePipelineTemplateV2(rawTemplate)
	if err != nil {
		return err
	}

	template.ID = data.Get("template_id").(string)

	if err := api.UpdatePipelineTemplateV2(client, template); err != nil {
		return fmt.Errorf("failed to update pipeline template %q: %w", template.ID, err)
	}

	data.SetId(template.ID)

	return resourcePipelineTemplateV2Read(data, meta)
}

func resourcePipelineTemplateV2Delete(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return err
	}

	templateID := data.Get("template_id").(string)

	versionMap, err := api.ListPipelineTemplateV2Versions(client)
	if err != nil {
		return fmt.Errorf("failed to list pipeline template versions: %w", err)
	}

	// Delete all versions for this pipeline template.
	for _, version := range versionMap[templateID] {
		err := api.DeletePipelineTemplateV2(client, version.ID, version.Tag, version.Digest)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pipeline template %q (tag: %q, digest: %q): %w",
				version.ID, version.Tag, version.Digest, err)
		}
	}

	data.SetId("")

	return nil
}

func resourcePipelineTemplateV2Exists(data *schema.ResourceData, meta interface{}) (bool, error) {
	clientConfig := meta.(*clientConfig)

	client, err := clientConfig.Client()
	if err != nil {
		return false, err
	}

	templateID := data.Get("template_id").(string)

	_, err = api.GetPipelineTemplateV2(client, templateID)
	if apierrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to fetch pipeline template %q: %w", templateID, err)
	}

	return true, nil
}

func parsePipelineTemplateV2(rawTemplate string) (*api.PipelineTemplateV2, error) {
	var template *api.PipelineTemplateV2

	if err := json.Unmarshal([]byte(rawTemplate), &template); err != nil {
		return nil, fmt.Errorf("invalid template json: %w", err)
	}

	if err := validatePipelineTemplateV2(template); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	return template, nil
}

func validatePipelineTemplateV2(template *api.PipelineTemplateV2) error {
	var errs *multierror.Error

	if template.ID != "" {
		errs = multierror.Append(errs, errors.New("field 'id' must not be set as they will be computed"))
	}

	if template.Schema != "v2" {
		errs = multierror.Append(errs, errors.New("field 'schema' must be set to 'v2'"))
	}

	if template.Pipeline == nil {
		errs = multierror.Append(errs, errors.New("field 'pipeline' missing"))
	}

	if template.Metadata.Name == "" {
		errs = multierror.Append(errs, errors.New("field 'metadata.name' must not be empty"))
	}

	if template.Metadata.Description == "" {
		errs = multierror.Append(errs, errors.New("field 'metadata.description' must not be empty"))
	}

	if len(template.Metadata.Scopes) == 0 {
		errs = multierror.Append(errs, errors.New("field 'metadata.scopes' must contain at least one scope"))
	}

	for i, variable := range template.Variables {
		if variable.Name == "" {
			errs = multierror.Append(errs, fmt.Errorf("field 'variables[%d].name' must not be empty", i))
		}

		if variable.Type == "" {
			errs = multierror.Append(errs, fmt.Errorf("field 'variables[%d].type' must not be empty", i))
		}
	}

	return errs.ErrorOrNil()
}
