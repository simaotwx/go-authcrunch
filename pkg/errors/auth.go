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

package errors

// Portal errors.
const (
	ErrStaticAssetAddFailed                  StandardError = "failed adding custom static asset %s (%s) from %s for %s portal: %v"
	ErrUserInterfaceThemeNotFound            StandardError = "user interface validation for %s portal failed: %s theme not found"
	ErrUserInterfaceBuiltinTemplateAddFailed StandardError = "user interface validation for %s portal failed for built-in template %s in %s theme: %v"
	ErrUserInterfaceCustomTemplateAddFailed  StandardError = "user interface validation for %s portal failed for custom template %s in %s: %v"

	ErrUserRegistrationConfig StandardError = "user registration configuration for %q instance failed: %v"
	ErrCryptoKeyStoreConfig   StandardError = "crypto key store configuration for %q instance failed: %v"
	ErrGeneric                StandardError = "%s: %v"

	ErrAuthorizationFailed StandardError = "user authorization failed: %s, reason: %v"
)
