# terraform-provider-spinnaker

[![Build Status](https://github.com/Bonial-International-GmbH/terraform-provider-spinnaker/workflows/build/badge.svg)](https://github.com/Bonial-International-GmbH/terraform-provider-spinnaker/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/Bonial-International-GmbH/terraform-provider-spinnaker?style=flat)](https://goreportcard.com/report/github.com/Bonial-International-GmbH/terraform-provider-spinnaker)
[![GoDoc](https://godoc.org/github.com/Bonial-International-GmbH/terraform-provider-spinnaker?status.svg)](https://godoc.org/github.com/Bonial-International-GmbH/terraform-provider-spinnaker)

Manage [Spinnaker](https://spinnaker.io) applications and pipelines with Terraform.

## Demo

![demo](https://d2ddoduugvun08.cloudfront.net/items/1A0A1C2C1M243j0b2u16/Screen%20Recording%202018-11-23%20at%2012.18%20PM.gif)

## Example

```
provider "spinnaker" {
  server = "http://spinnaker-gate.myorg.io"
}

resource "spinnaker_application" "my_app" {
  application = "terraformtest"
  email       = "ethan@armory.io"
}

resource "spinnaker_pipeline" "terraform_example" {
  application = spinnaker_application.my_app.application
  name        = "Example Pipeline"
  pipeline    = file("pipelines/example.json")
}
```

## Installation

#### Build from Source

_Requires Go to be installed on the system._

```
$ env GO111MODULE=on go get github.com/armory-io/terraform-provider-spinnaker
$ cd $GOPATH/src/github.com/armory-io/terraform-provider-spinnaker
$ env GO111MODULE=on go build
```

#### Installing 3rd Party Plugins

See [Terraform documentation](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins) for installing 3rd party plugins.

## Provider

#### Example Usage

```
provider "spinnaker" {
  server             = "http://spinnaker-gate.myorg.io"
  config             = "/path/to/config.yml"
  ignore_cert_errors = true
  default_headers    = "Api-Key=abc123"
}
```

#### Argument Reference

Every argument can also be set via the environment variable listed alongside it.
Values set in the provider block take precedence over the environment.

* `server` - (`GATE_URL`) The Gate API Url
* `config` - (Optional) - (`SPINNAKER_CONFIG_PATH`) Path to Gate config file. See the [Spin CLI](https://github.com/spinnaker/spin/blob/master/config/example.yaml) for an example config. The file is optional and only read for authentication settings that are not provided via the `oauth2_*` arguments below.
* `ignore_cert_errors` - (Optional) - Set this to `true` to ignore certificate errors from Gate. Defaults to `false`.
* `default_headers` - (Optional) - Pass through a comma separated set of key value pairs to set default headers for the gate client when sending requests to your gate endpoint e.g. "header1=value1,header2=value2". Defaults to "".
* `oauth2_client_id` - (Optional) - (`SPINNAKER_OAUTH2_CLIENT_ID`) OAuth2 client ID.
* `oauth2_client_secret` - (Optional) - (`SPINNAKER_OAUTH2_CLIENT_SECRET`) OAuth2 client secret.
* `oauth2_token_url` - (Optional) - (`SPINNAKER_OAUTH2_TOKEN_URL`) OAuth2 token endpoint. Defaults to the `/oauth2/token` endpoint of `server`.
* `oauth2_scope` - (Optional) - (`SPINNAKER_OAUTH2_SCOPE`) Space separated list of OAuth2 scopes to request, e.g. `"scope1 scope2"`. Defaults to requesting no scopes.

#### Authentication

Setting both `oauth2_client_id` and `oauth2_client_secret` enables authentication
via the [OAuth2 client credentials flow](https://oauth.net/2/grant-types/client-credentials/).
Setting only one of the two is an error. Access tokens are fetched on demand and
refreshed automatically once they expire.

```
provider "spinnaker" {
  server               = "http://spinnaker-gate.myorg.io"
  oauth2_client_id     = var.spinnaker_client_id
  oauth2_client_secret = var.spinnaker_client_secret
  oauth2_token_url     = "https://myorg.auth.eu-central-1.amazoncognito.com/oauth2/token"
  oauth2_scope         = "spinnaker/read spinnaker/write"
}
```

If the `oauth2_*` arguments are not set, authentication falls back to the `auth`
section of the Gate config file, which supports the same mechanisms as the Spin
CLI (x509, basic, LDAP, IAP, Google service accounts and the OAuth2
authorization code flow). Note that the interactive mechanisms are a poor fit
for a Terraform provider, as they prompt on stdin.

## Resources

### `spinnaker_application`

#### Example Usage

```
resource "spinnaker_application" "my_app" {
  application = "terraformtest"
  email       = "ethan@armory.io"
}
```
#### Argument Reference
* `application` - Application name
* `email` - Owner email

### `spinnaker_pipeline`

#### Example Usage

```
resource "spinnaker_pipeline" "terraform_example" {
  application = spinnaker_application.my_app.application
  name        = "Example Pipeline"
  pipeline    = file("pipelines/example.json")
}
```

#### Argument Reference

* `application` - Application name
* `name` - Pipeline name
* `pipeline` - Pipeline JSON in string format, example `file(pipelines/example.json)`

### `spinnaker_pipeline_template`

#### Example Usage

```
resource "spinnaker_pipeline_template" "terraform_example" {
  template = templatefile("template.yml", {
    yourVariable = var.your-variable
    })
}
```

#### Argument Reference

* `template` - A yaml formated [DCD Spec pipeline template](https://github.com/spinnaker/dcd-spec/blob/master/PIPELINE_TEMPLATES.md#templates) 

### `spinnaker_pipeline_template_config`

#### Example Usage

```
resource "spinnaker_pipeline_template_config" "terraform_example" {
  pipeline_config = templatefile("config.yml",{
    yourVariable = var.your-variable
  })
}
```

#### Argument Reference

* `pipeline_config` - A yaml formated [DCD Spec pipeline configuration](https://github.com/spinnaker/dcd-spec/blob/master/PIPELINE_TEMPLATES.md#configurations)
