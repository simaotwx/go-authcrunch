// Copyright 2022 Paul Greenberg greenpau@outlook.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oauth2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/greenpau/go-authcrunch/pkg/errors"
	"go.uber.org/zap"
)

// Config holds the configuration for the backend.
type Config struct {
	Name     string `json:"name,omitempty" xml:"name,omitempty" yaml:"name,omitempty"`
	Method   string `json:"method,omitempty" xml:"method,omitempty" yaml:"method,omitempty"`
	Realm    string `json:"realm,omitempty" xml:"realm,omitempty" yaml:"realm,omitempty"`
	Provider string `json:"provider,omitempty" xml:"provider,omitempty" yaml:"provider,omitempty"`

	DomainName        string `json:"domain_name,omitempty" xml:"domain_name,omitempty" yaml:"domain_name,omitempty"`
	ClientID          string `json:"client_id,omitempty" xml:"client_id,omitempty" yaml:"client_id,omitempty"`
	ClientSecret      string `json:"client_secret,omitempty" xml:"client_secret,omitempty" yaml:"client_secret,omitempty"`
	ServerID          string `json:"server_id,omitempty" xml:"server_id,omitempty" yaml:"server_id,omitempty"`
	AppSecret         string `json:"app_secret,omitempty" xml:"app_secret,omitempty" yaml:"app_secret,omitempty"`
	TenantID          string `json:"tenant_id,omitempty" xml:"tenant_id,omitempty" yaml:"tenant_id,omitempty"`
	IdentityTokenName string `json:"identity_token_name,omitempty" xml:"identity_token_name,omitempty" yaml:"identity_token_name,omitempty"`

	Scopes []string `json:"scopes,omitempty" xml:"scopes,omitempty" yaml:"scopes,omitempty"`

	// The number if seconds to wait before getting key material
	// from an OAuth 2.0 backend.
	DelayStart int `json:"delay_start,omitempty" xml:"delay_start,omitempty" yaml:"delay_start,omitempty"`
	// The number of the retry attempts getting key material
	// from an OAuth 2.0 backend.
	RetryAttempts int `json:"retry_attempts,omitempty" xml:"retry_attempts,omitempty" yaml:"retry_attempts,omitempty"`
	// The number of seconds to wait until the retrying.
	RetryInterval int `json:"retry_interval,omitempty" xml:"retry_interval,omitempty" yaml:"retry_interval,omitempty"`

	UserRoleMapList []map[string]interface{} `json:"user_roles,omitempty" xml:"user_roles,omitempty" yaml:"user_roles,omitempty"`

	// The URL to OAuth 2.0 Custom Authorization Server.
	BaseAuthURL string `json:"base_auth_url,omitempty" xml:"base_auth_url,omitempty" yaml:"base_auth_url,omitempty"`

	// The URL to OAuth 2.0 metadata related to your Custom Authorization Server.
	MetadataURL string `json:"metadata_url,omitempty" xml:"metadata_url,omitempty" yaml:"metadata_url,omitempty"`

	// The regex filters for user groups extracted via IdP API.
	UserGroupFilters []string `json:"user_group_filters,omitempty" xml:"user_group_filters,omitempty" yaml:"user_group_filters,omitempty"`
	// The regex filters for user orgs extracted via IdP API.
	UserOrgFilters []string `json:"user_org_filters,omitempty" xml:"user_org_filters,omitempty" yaml:"user_org_filters,omitempty"`

	// Disabled metadata discovery via public metadata URL.
	MetadataDiscoveryDisabled bool `json:"metadata_discovery_disabled,omitempty" xml:"metadata_discovery_disabled,omitempty" yaml:"metadata_discovery_disabled,omitempty"`

	KeyVerificationDisabled bool `json:"key_verification_disabled,omitempty" xml:"key_verification_disabled,omitempty" yaml:"key_verification_disabled,omitempty"`
	PassGrantTypeDisabled   bool `json:"pass_grant_type_disabled,omitempty" xml:"pass_grant_type_disabled,omitempty" yaml:"pass_grant_type_disabled,omitempty"`
	ResponseTypeDisabled    bool `json:"response_type_disabled,omitempty" xml:"response_type_disabled,omitempty" yaml:"response_type_disabled,omitempty"`
	NonceDisabled           bool `json:"nonce_disabled,omitempty" xml:"nonce_disabled,omitempty" yaml:"nonce_disabled,omitempty"`
	ScopeDisabled           bool `json:"scope_disabled,omitempty" xml:"scope_disabled,omitempty" yaml:"scope_disabled,omitempty"`

	AcceptHeaderEnabled bool `json:"accept_header_enabled,omitempty" xml:"accept_header_enabled,omitempty" yaml:"accept_header_enabled,omitempty"`

	JsCallbackEnabled bool `json:"js_callback_enabled,omitempty" xml:"js_callback_enabled,omitempty" yaml:"js_callback_enabled,omitempty"`

	ResponseType []string `json:"response_type,omitempty" xml:"response_type,omitempty" yaml:"response_type,omitempty"`

	AuthorizationURL string `json:"authorization_url,omitempty" xml:"authorization_url,omitempty" yaml:"authorization_url,omitempty"`
	TokenURL         string `json:"token_url,omitempty" xml:"token_url,omitempty" yaml:"token_url,omitempty"`

	RequiredTokenFields []string `json:"required_token_fields,omitempty" xml:"required_token_fields,omitempty" yaml:"required_token_fields,omitempty"`

	scopeMap map[string]interface{}
}

// ScopeExists returns true if any of the provided scopes exist.
func (c *Config) ScopeExists(scopes ...string) bool {
	for _, scope := range scopes {
		if _, exists := c.scopeMap[scope]; exists {
			return true
		}
	}
	return false
}

// Configure configures Backend.
func (b *Backend) Configure() error {
	if b.Config.Name == "" {
		return errors.ErrBackendConfigureNameEmpty
	}
	if b.Config.Method == "" {
		return errors.ErrBackendConfigureMethodEmpty
	}
	if b.Config.Realm == "" {
		return errors.ErrBackendConfigureRealmEmpty
	}

	if b.Config.ClientID == "" {
		return errors.ErrBackendClientIDNotFound.WithArgs(b.Config.Provider)
	}
	if b.Config.ClientSecret == "" {
		return errors.ErrBackendClientSecretNotFound.WithArgs(b.Config.Provider)
	}

	if b.Config.DelayStart > 0 {
		if b.Config.RetryAttempts < 1 {
			b.Config.RetryAttempts = 2
		}
		if b.Config.RetryInterval == 0 {
			b.Config.RetryInterval = b.Config.DelayStart
		}
		b.logger.Debug(
			"Configured delayed start for OAuth 2.0 backend",
			zap.String("backend_name", b.Config.Name),
			zap.Int("delayed_by", b.Config.DelayStart),
			zap.Int("retry_attempts", b.Config.RetryAttempts),
			zap.Int("retry_interval", b.Config.RetryInterval),
		)
	}

	if b.Config.RetryAttempts > 0 && b.Config.DelayStart == 0 {
		if b.Config.RetryInterval == 0 {
			b.Config.RetryInterval = 5
		}
		b.logger.Debug(
			"Configured connection retries for OAuth 2.0 backend",
			zap.String("backend_name", b.Config.Name),
			zap.Int("retry_attempts", b.Config.RetryAttempts),
			zap.Int("retry_interval", b.Config.RetryInterval),
		)
	}

	if len(b.Config.Scopes) < 1 {
		switch b.Config.Provider {
		case "facebook":
			b.Config.Scopes = []string{"public_profile", "email"}
		case "nextcloud":
			b.Config.Scopes = []string{"email"}
		default:
			b.Config.Scopes = []string{"openid", "email", "profile"}
		}
	}

	if b.Config.KeyVerificationDisabled {
		b.disableKeyVerification = true
	}
	if b.Config.PassGrantTypeDisabled {
		b.disablePassGrantType = true
	}
	if b.Config.ResponseTypeDisabled {
		b.disableResponseType = true
	}
	if b.Config.NonceDisabled {
		b.disableNonce = true
	}
	if b.Config.ScopeDisabled {
		b.disableScope = true
	}

	if b.Config.AcceptHeaderEnabled {
		b.enableAcceptHeader = true
	}

	if b.Config.AuthorizationURL != "" {
		b.authorizationURL = b.Config.AuthorizationURL
	}
	if b.Config.TokenURL != "" {
		b.tokenURL = b.Config.TokenURL
	}

	b.requiredTokenFields = make(map[string]interface{})
	for _, fieldName := range b.Config.RequiredTokenFields {
		b.requiredTokenFields[fieldName] = true
	}

	b.Config.scopeMap = make(map[string]interface{})
	for _, scope := range b.Config.Scopes {
		b.Config.scopeMap[scope] = true
	}

	switch b.Config.IdentityTokenName {
	case "":
		b.Config.IdentityTokenName = "id_token"
	case "id_token", "access_token":
	default:
		return errors.ErrBackendInvalidIdentityTokenName.WithArgs(b.Config.IdentityTokenName, b.Config.Provider)
	}

	switch b.Config.Provider {
	case "okta":
		if b.Config.ServerID == "" {
			return errors.ErrBackendServerIDNotFound.WithArgs(b.Config.Provider)
		}
		if b.Config.DomainName == "" {
			return errors.ErrBackendAppNameNotFound.WithArgs(b.Config.Provider)
		}
		if b.Config.BaseAuthURL == "" {
			b.Config.BaseAuthURL = fmt.Sprintf(
				"https://%s/oauth2/%s/",
				b.Config.DomainName, b.Config.ServerID,
			)
			b.Config.MetadataURL = b.Config.BaseAuthURL + ".well-known/openid-configuration?client_id=" + b.Config.ClientID
		}
	case "google":
		if b.Config.BaseAuthURL == "" {
			b.Config.BaseAuthURL = "https://accounts.google.com/o/oauth2/v2/"
			b.Config.MetadataURL = "https://accounts.google.com/.well-known/openid-configuration"
		}
		// If Google client_id does not contains domain name, append with
		// the default of .apps.googleusercontent.com.
		if !strings.Contains(b.Config.ClientID, ".") {
			b.Config.ClientID = b.Config.ClientID + ".apps.googleusercontent.com"
		}
	case "github":
		if b.Config.BaseAuthURL == "" {
			b.Config.BaseAuthURL = "https://github.com/login/oauth/"
		}
		b.authorizationURL = "https://github.com/login/oauth/authorize"
		b.tokenURL = "https://github.com/login/oauth/access_token"
		b.disableKeyVerification = true
		b.disablePassGrantType = true
		b.disableResponseType = true
		b.disableNonce = true
		b.enableAcceptHeader = true
		b.requiredTokenFields = map[string]interface{}{
			"access_token": true,
		}
	case "gitlab":
		if b.Config.DomainName == "" {
			b.Config.DomainName = "gitlab.com"
		}
		if b.Config.BaseAuthURL == "" {
			b.Config.BaseAuthURL = fmt.Sprintf("https://%s/", b.Config.DomainName)
			b.Config.MetadataURL = b.Config.BaseAuthURL + ".well-known/openid-configuration"
		}
	case "azure":
		if b.Config.TenantID == "" {
			b.Config.TenantID = "common"
		}
		if b.Config.BaseAuthURL == "" {
			b.Config.BaseAuthURL = "https://login.microsoftonline.com/" + b.Config.TenantID + "/oauth2/v2.0/"
			b.Config.MetadataURL = "https://login.microsoftonline.com/" + b.Config.TenantID + "/v2.0/.well-known/openid-configuration"
		}
	case "facebook":
		if b.Config.BaseAuthURL == "" {
			b.Config.BaseAuthURL = "https://www.facebook.com/v12.0/dialog/"
		}
		b.authorizationURL = "https://www.facebook.com/v12.0/dialog/oauth"
		b.tokenURL = "https://graph.facebook.com/v12.0/oauth/access_token"
		b.disableKeyVerification = true
		b.disablePassGrantType = true
		b.disableResponseType = true
		b.disableNonce = true
		b.enableAcceptHeader = true
		b.requiredTokenFields = map[string]interface{}{
			"access_token": true,
		}
	case "nextcloud":
		b.authorizationURL = fmt.Sprintf("%s/apps/oauth2/authorize", b.Config.BaseAuthURL)
		b.tokenURL = fmt.Sprintf("%s/apps/oauth2/api/v1/token", b.Config.BaseAuthURL)
		b.disableKeyVerification = true
	case "generic":
	case "":
		return errors.ErrBackendOauthProviderNotFound.WithArgs(b.Config.Provider)
	default:
		return errors.ErrBackendUnsupportedProvider.WithArgs(b.Config.Provider)
	}

	if b.Config.BaseAuthURL == "" {
		return errors.ErrBackendOauthAuthorizationURLNotFound.WithArgs(b.Config.Provider)
	}

	parsedBaseAuthURL, err := url.Parse(b.Config.BaseAuthURL)
	if err != nil {
		return errors.ErrBackendConfigureInvalidBaseURL.WithArgs(b.Config.Provider, b.Config.BaseAuthURL, err)
	}
	b.serverName = parsedBaseAuthURL.Host

	if len(b.requiredTokenFields) < 1 {
		for _, fieldName := range []string{"access_token", "id_token"} {
			b.requiredTokenFields[fieldName] = true
			b.Config.RequiredTokenFields = append(b.Config.RequiredTokenFields, fieldName)
		}
	}

	if len(b.Config.ResponseType) < 1 {
		b.Config.ResponseType = []string{"code"}
	}

	if b.Config.DelayStart > 0 {
		go b.fetchConfig()
	} else {
		if err := b.fetchConfig(); err != nil {
			return err
		}
	}

	// Configure user group filters, if any.
	for _, pattern := range b.Config.UserGroupFilters {
		compiledPattern, err := regexp.Compile(pattern)
		if err != nil {
			return errors.ErrBackendOAuthUserGroupFilterInvalid.WithArgs(pattern, err)
		}
		b.userGroupFilters = append(b.userGroupFilters, compiledPattern)
	}

	// Configure user org filters, if any.
	for _, pattern := range b.Config.UserOrgFilters {
		compiledPattern, err := regexp.Compile(pattern)
		if err != nil {
			return errors.ErrBackendOAuthUserOrgFilterInvalid.WithArgs(pattern, err)
		}
		b.userOrgFilters = append(b.userOrgFilters, compiledPattern)
	}

	b.logger.Info(
		"successfully configured OAuth 2.0 backend",
		zap.String("provider", b.Config.Provider),
		zap.String("client_id", b.Config.ClientID),
		zap.String("server_id", b.Config.ServerID),
		zap.String("domain_name", b.Config.DomainName),
		zap.Any("metadata", b.metadata),
		zap.Any("jwks_keys", b.keys),
		zap.Strings("required_token_fields", b.Config.RequiredTokenFields),
	)

	return nil
}

func (b *Backend) fetchConfig() error {
	if b.Config.DelayStart > 0 {
		b.logger.Debug(
			"Delaying backend configuration",
			zap.String("backend_name", b.Config.Name),
			zap.Int("delayed_by", b.Config.DelayStart),
		)
		time.Sleep(time.Duration(b.Config.DelayStart) * time.Second)
	}

	if b.authorizationURL == "" {
		if b.Config.RetryAttempts > 0 {
			for i := 0; i < b.Config.RetryAttempts; i++ {
				err := b.fetchMetadataURL()
				if err == nil {
					break
				}
				if i >= (b.Config.RetryAttempts - 1) {
					return errors.ErrBackendOauthMetadataFetchFailed.WithArgs(err)
				}
				b.logger.Debug(
					"fetchMetadataURL failed",
					zap.String("backend_name", b.Config.Name),
					zap.Int("attempt_id", i),
					zap.Error(errors.ErrBackendOauthMetadataFetchFailed.WithArgs(err)),
				)
				time.Sleep(time.Duration(b.Config.RetryInterval) * time.Second)
			}
		} else {
			if err := b.fetchMetadataURL(); err != nil {
				b.logger.Debug(
					"fetchMetadataURL failed",
					zap.String("backend_name", b.Config.Name),
					zap.Error(errors.ErrBackendOauthMetadataFetchFailed.WithArgs(err)),
				)
				return errors.ErrBackendOauthMetadataFetchFailed.WithArgs(err)
			}
		}
		b.logger.Debug(
			"fetchMetadataURL succeeded",
			zap.String("backend_name", b.Config.Name),
			zap.Any("metadata", b.metadata),
			zap.Any("userinfo_endpoint", b.userInfoURL),
		)
	}

	if !b.disableKeyVerification {
		if b.Config.RetryAttempts > 0 {
			for i := 0; i < b.Config.RetryAttempts; i++ {
				err := b.fetchKeysURL()
				if err == nil {
					break
				}
				if i >= (b.Config.RetryAttempts - 1) {
					return errors.ErrBackendOauthKeyFetchFailed.WithArgs(err)
				}
				b.logger.Debug(
					"fetchKeysURL failed",
					zap.String("backend_name", b.Config.Name),
					zap.Int("attempt_id", i),
					zap.Error(errors.ErrBackendOauthKeyFetchFailed.WithArgs(err)),
				)
				time.Sleep(time.Duration(b.Config.RetryInterval) * time.Second)
			}
		} else {
			if err := b.fetchKeysURL(); err != nil {
				return errors.ErrBackendOauthKeyFetchFailed.WithArgs(err)
			}
		}
	}
	return nil
}

func (b *Backend) fetchMetadataURL() error {
	cli, err := newBrowser()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", b.Config.MetadataURL, nil)
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, &b.metadata); err != nil {
		return err
	}
	for _, k := range []string{"authorization_endpoint", "token_endpoint", "jwks_uri"} {
		if _, exists := b.metadata[k]; !exists {
			return errors.ErrBackendOauthMetadataFieldNotFound.WithArgs(k, b.Config.Provider)
		}
	}
	b.authorizationURL = b.metadata["authorization_endpoint"].(string)
	b.tokenURL = b.metadata["token_endpoint"].(string)
	b.keysURL = b.metadata["jwks_uri"].(string)
	if _, exists := b.metadata["userinfo_endpoint"]; exists {
		b.userInfoURL = b.metadata["userinfo_endpoint"].(string)
	}
	return nil
}

func (b *Backend) countFetchKeysAttempt() {
	b.lastKeyFetch = time.Now().UTC()
	b.keyFetchAttempts++
	return
}

func (b *Backend) fetchKeysURL() error {
	if b.keyFetchAttempts > 3 {
		timeDiff := time.Now().UTC().Sub(b.lastKeyFetch).Minutes()
		if timeDiff < 5 {
			return errors.ErrBackendOauthJwksKeysTooManyAttempts
		}
		b.lastKeyFetch = time.Now().UTC()
		b.keyFetchAttempts = 0
	}
	b.countFetchKeysAttempt()
	resp, err := http.Get(b.keysURL)
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	data := make(map[string]interface{})

	if err := json.Unmarshal(respBody, &data); err != nil {
		return err
	}

	if _, exists := data["keys"]; !exists {
		return errors.ErrBackendOauthJwksResponseKeysNotFound
	}

	jwksJSON, err := json.Marshal(data["keys"])
	if err != nil {
		return errors.ErrBackendOauthJwksKeysParseFailed.WithArgs(err)
	}

	keys := []*JwksKey{}
	if err := json.Unmarshal(jwksJSON, &keys); err != nil {
		return err
	}

	if len(keys) < 1 {
		return errors.ErrBackendOauthJwksKeysNotFound
	}

	for _, k := range keys {
		if err := k.Validate(); err != nil {
			return errors.ErrBackendOauthJwksInvalidKey.WithArgs(err)
		}
		b.keys[k.KeyID] = k
	}

	return nil
}

// Validate checks whether Backend is functional.
func (b *Backend) Validate() error {
	return nil
}

// GetConfig returns Backend configuration.
func (b *Backend) GetConfig() string {
	var sb strings.Builder
	sb.WriteString("name " + b.Config.Name + "\n")
	sb.WriteString("method " + b.Config.Method + "\n")
	sb.WriteString("realm " + b.Config.Realm + "\n")
	sb.WriteString("provider " + b.Config.Provider)
	return sb.String()
}
