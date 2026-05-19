package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var OneLoginClientId = field.StringField(
	"onelogin-client-id",
	field.WithDisplayName("Client ID"),
	field.WithDescription("OneLogin client ID used to generate the access token."),
	field.WithRequired(true),
)

var OneLoginClientSecret = field.StringField(
	"onelogin-client-secret",
	field.WithDisplayName("Client Secret"),
	field.WithDescription("OneLogin client secret used to generate the access token"),
	field.WithRequired(true),
	field.WithIsSecret(true),
)

var OneLoginSubDomain = field.StringField(
	"subdomain",
	field.WithDisplayName("One Login Subdomain"),
	field.WithDescription("OneLogin subdomain to connect to"),
	field.WithRequired(true),
)

var OneLoginEnablePrivileges = field.BoolField(
	"privileges-enabled",
	field.WithDisplayName("Enable Privileges sync"),
	field.WithDescription("Enable syncing of privileges from OneLogin. Requires OneLogin subscription to have access to privileges."),
	field.WithDefaultValue(true),
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{
		OneLoginClientId,
		OneLoginClientSecret,
		OneLoginSubDomain,
		OneLoginEnablePrivileges,
	},
	field.WithConnectorDisplayName("OneLogin"),
	field.WithHelpUrl("/docs/baton/onelogin-v2"),
	field.WithIconUrl("/static/app-icons/onelogin.svg"),
)

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(cfg *Onelogin) error {
	return nil
}
