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

package handlers

import (
	"github.com/greenpau/go-authcrunch/pkg/requests"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

var jsRedirTmpl = template.Must(template.New("js_redir").Parse(`
<html>
    <body>
        <p>User Unauthorized. Redirecting to login.</p>
        <script>
        var auth_url_path = "{{.AuthURLPath}}";
        var sep = "{{.Sep}}";
        var redir_param = "{{.RedirParam}}";
        var redir_url = "{{.RedirURL}}";
        if (window.location.hash) {
            redir_url = redir_url + "#" + window.location.hash.substr(1);
        }
        var final_url = auth_url_path;
        if (redir_param) {
            final_url = auth_url_path + sep + redir_param + "=" + encodeURIComponent(redir_url);
        }
        window.location = final_url;
        </script>
    </body>
</html>
`))

// HandleLocationHeaderRedirect redirects the requests to configured auth URL
// by setting Location header and sending 302.
func HandleLocationHeaderRedirect(w http.ResponseWriter, r *http.Request, rr *requests.AuthorizationRequest) {
	configureRedirect(w, r, rr)
	if !rr.Redirect.Enabled {
		return
	}

	if rr.Redirect.QueryDisabled {
		w.Header().Set("Location", rr.Redirect.AuthURL)
	} else {
		var sb strings.Builder
		sb.WriteString(rr.Redirect.AuthURL)
		sb.WriteString(rr.Redirect.Separator)
		sb.WriteString(rr.Redirect.QueryParameter)
		sb.WriteString("=")
		sb.WriteString(url.QueryEscape(rr.Redirect.URL))
		w.Header().Set("Location", sb.String())
	}

	if rr.Redirect.StatusCode == 0 {
		rr.Redirect.StatusCode = 302
	}

	w.WriteHeader(rr.Redirect.StatusCode)
	w.Write([]byte(http.StatusText(rr.Redirect.StatusCode)))
	return
}

// HandleJavascriptRedirect redirects the requests to configured auth URL by
// responding Javascript-enabled HTML performing script-based redirection.
func HandleJavascriptRedirect(w http.ResponseWriter, r *http.Request, rr *requests.AuthorizationRequest) {
	configureRedirect(w, r, rr)
	if !rr.Redirect.Enabled {
		return
	}

	if rr.Redirect.StatusCode == 0 {
		rr.Redirect.StatusCode = 401
	}

	w.WriteHeader(rr.Redirect.StatusCode)
	jsRedirTmpl.Execute(w, map[string]string{
		"AuthURLPath": rr.Redirect.AuthURL,
		"Sep":         rr.Redirect.Separator,
		"RedirParam":  rr.Redirect.QueryParameter,
		"RedirURL":    rr.Redirect.URL,
	})
	return
}

func configureRedirect(w http.ResponseWriter, r *http.Request, rr *requests.AuthorizationRequest) {
	if strings.Contains(r.RequestURI, rr.Redirect.QueryParameter) {
		rr.Redirect.Enabled = false
		return
	}

	rr.Redirect.Enabled = true

	if rr.Redirect.QueryDisabled {
		return
	}

	rr.Redirect.Separator = "?"
	rr.Redirect.URL = r.RequestURI

	if strings.HasPrefix(rr.Redirect.URL, "/") {
		redirHost := r.Header.Get("X-Forwarded-Host")
		if redirHost == "" {
			redirHost = r.Host
		}
		redirProto := r.Header.Get("X-Forwarded-Proto")
		if redirProto == "" {
			if r.TLS == nil {
				redirProto = "http"
			} else {
				redirProto = "https"
			}
		}
		redirPort := r.Header.Get("X-Forwarded-Port")

		redirectBaseURL := redirProto + "://" + redirHost
		if redirPort != "" {
			switch redirPort {
			case "443":
				if redirProto != "https" {
					redirectBaseURL += ":" + redirPort
				}
			case "80":
				if redirProto != "http" {
					redirectBaseURL += ":" + redirPort
				}
			default:
				redirectBaseURL += ":" + redirPort
			}
		}
		rr.Redirect.URL = redirectBaseURL + r.RequestURI
	}

	if strings.Contains(rr.Redirect.AuthURL, "?") {
		rr.Redirect.Separator = "&"
	}

	if rr.Redirect.LoginHint != "" {
		loginHint := rr.Redirect.LoginHint
		escapedLoginHint := url.QueryEscape(loginHint)
		rr.Redirect.AuthURL = rr.Redirect.AuthURL + rr.Redirect.Separator + "login_hint" + "=" + escapedLoginHint
		rr.Redirect.Separator = "&"
	}

	return
}
