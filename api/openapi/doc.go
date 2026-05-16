// Package openapidoc contains the global OpenAPI metadata for the Fieldstone
// public gateway API. Run `make swagger` to regenerate the spec.
//
// @title           Fieldstone Civic Services API
// @version         0.1.0
// @description     A self-hostable platform for city governments to run common civic services.
// @description
// @description     Staff endpoints require a Bearer JWT from the configured OIDC provider.
// @description     Citizen endpoints (marked public) do not require authentication.
// @description
// @description     **Development note**: when `DEV_DISABLE_AUTH=true` all requests are treated
// @description     as authenticated. Remove this flag before production deployment.
//
// @contact.name    Fieldstone Contributors
// @contact.url     https://github.com/fieldstone/fieldstone
//
// @license.name    MIT
// @license.url     https://github.com/fieldstone/fieldstone/blob/main/LICENSE
//
// @host            localhost:8080
// @BasePath        /
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT Bearer token from your OIDC provider: `Bearer <token>`
package openapidoc
