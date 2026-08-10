package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
)

const (
	ErrCodeNoSuchEntityException = "NoSuchEntityException"
)

func CreatePipelineTemplate(client *Client, template interface{}) error {
	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		resp, err := client.PipelineTemplatesControllerApi.CreateUsingPOST(client.Context, template)

		return nil, resp, err
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("encountered an error saving template, status code: %d", resp.StatusCode)
	}

	return nil
}

func GetPipelineTemplate(client *Client, templateID string, dest interface{}) error {
	successPayload, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		return client.PipelineTemplatesControllerApi.GetUsingGET(client.Context, templateID)
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%s", ErrCodeNoSuchEntityException)
		}
		return fmt.Errorf("encountered an error getting pipeline template %s, %s",
			templateID,
			err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("encountered an error getting pipeline template %s, status code: %d",
			templateID,
			resp.StatusCode,
		)
	}

	if successPayload == nil {
		return errors.New(ErrCodeNoSuchEntityException)
	}

	if err := mapstructure.Decode(successPayload, dest); err != nil {
		return err
	}

	return nil
}

func DeletePipelineTemplate(client *Client, templateID string) error {
	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		return client.PipelineTemplatesControllerApi.DeleteUsingDELETE(client.Context, templateID, nil)
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("encountered an error deleting pipeline template %s, status code: %d",
			templateID,
			resp.StatusCode)
	}

	return nil
}

func UpdatePipelineTemplate(client *Client, templateID string, template interface{}) error {
	_, resp, err := retry(func() (map[string]interface{}, *http.Response, error) {
		resp, err := client.PipelineTemplatesControllerApi.UpdateUsingPOST(client.Context, templateID, template, nil)

		return nil, resp, err
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("encountered an error updating pipeline template %s, status code: %d",
			templateID,
			resp.StatusCode)
	}

	return nil
}
