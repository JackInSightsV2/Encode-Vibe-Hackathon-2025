package auth

// Predefined OAuth2 provider configurations for common identity providers

// GoogleProvider returns a pre-configured Google OAuth2 provider
func GoogleProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "google",
		Name:         "Google",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		DiscoveryURL: "https://accounts.google.com/.well-known/openid_configuration",
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "email", // Google uses email as username
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// MicrosoftProvider returns a pre-configured Microsoft OAuth2 provider
func MicrosoftProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "microsoft",
		Name:         "Microsoft",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		UserInfoURL:  "https://graph.microsoft.com/v1.0/me",
		DiscoveryURL: "https://login.microsoftonline.com/common/v2.0/.well-known/openid_configuration",
		Scopes:       []string{"openid", "profile", "email", "User.Read"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "mail",
			UsernameField: "userPrincipalName",
			NameField:     "displayName",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// GitHubProvider returns a pre-configured GitHub OAuth2 provider
func GitHubProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "github",
		Name:         "GitHub",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       []string{"user:email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "login",
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// GitLabProvider returns a pre-configured GitLab OAuth2 provider
func GitLabProvider(clientID, clientSecret, redirectURL string, gitlabURL ...string) *OAuth2Provider {
	baseURL := "https://gitlab.com"
	if len(gitlabURL) > 0 && gitlabURL[0] != "" {
		baseURL = gitlabURL[0]
	}

	return &OAuth2Provider{
		ID:           "gitlab",
		Name:         "GitLab",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      baseURL + "/oauth/authorize",
		TokenURL:     baseURL + "/oauth/token",
		UserInfoURL:  baseURL + "/api/v4/user",
		Scopes:       []string{"read_user"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "username",
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// LinkedInProvider returns a pre-configured LinkedIn OAuth2 provider
func LinkedInProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "linkedin",
		Name:         "LinkedIn",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://www.linkedin.com/oauth/v2/authorization",
		TokenURL:     "https://www.linkedin.com/oauth/v2/accessToken",
		UserInfoURL:  "https://api.linkedin.com/v2/people/~:(id,firstName,lastName,emailAddress)",
		Scopes:       []string{"r_liteprofile", "r_emailaddress"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "emailAddress",
			UsernameField: "emailAddress",
			NameField:     "firstName.localized.en_US", // LinkedIn has complex name structure
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// SlackProvider returns a pre-configured Slack OAuth2 provider
func SlackProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "slack",
		Name:         "Slack",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://slack.com/oauth/v2/authorize",
		TokenURL:     "https://slack.com/api/oauth.v2.access",
		UserInfoURL:  "https://slack.com/api/users.identity",
		Scopes:       []string{"identity.basic", "identity.email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "user.email",
			UsernameField: "user.name",
			NameField:     "user.real_name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// DiscordProvider returns a pre-configured Discord OAuth2 provider
func DiscordProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "discord",
		Name:         "Discord",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://discord.com/api/oauth2/authorize",
		TokenURL:     "https://discord.com/api/oauth2/token",
		UserInfoURL:  "https://discord.com/api/users/@me",
		Scopes:       []string{"identify", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "username",
			NameField:     "global_name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// AppleProvider returns a pre-configured Apple OAuth2 provider
func AppleProvider(clientID, clientSecret, redirectURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "apple",
		Name:         "Apple",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://appleid.apple.com/auth/authorize",
		TokenURL:     "https://appleid.apple.com/auth/token",
		UserInfoURL:  "", // Apple doesn't have a traditional userinfo endpoint
		DiscoveryURL: "https://appleid.apple.com/.well-known/openid_configuration",
		Scopes:       []string{"name", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "email",
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// OktaProvider returns a pre-configured Okta OAuth2 provider
func OktaProvider(clientID, clientSecret, redirectURL, oktaDomain string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "okta",
		Name:         "Okta",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      oktaDomain + "/oauth2/default/v1/authorize",
		TokenURL:     oktaDomain + "/oauth2/default/v1/token",
		UserInfoURL:  oktaDomain + "/oauth2/default/v1/userinfo",
		DiscoveryURL: oktaDomain + "/oauth2/default/.well-known/openid_configuration",
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "preferred_username",
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// Auth0Provider returns a pre-configured Auth0 OAuth2 provider
func Auth0Provider(clientID, clientSecret, redirectURL, auth0Domain string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           "auth0",
		Name:         "Auth0",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      auth0Domain + "/authorize",
		TokenURL:     auth0Domain + "/oauth/token",
		UserInfoURL:  auth0Domain + "/userinfo",
		DiscoveryURL: auth0Domain + "/.well-known/openid_configuration",
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "nickname",
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// KeycloakProvider returns a pre-configured Keycloak OAuth2 provider
func KeycloakProvider(clientID, clientSecret, redirectURL, keycloakURL, realm string) *OAuth2Provider {
	baseURL := keycloakURL + "/realms/" + realm
	
	return &OAuth2Provider{
		ID:           "keycloak",
		Name:         "Keycloak",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      baseURL + "/protocol/openid-connect/auth",
		TokenURL:     baseURL + "/protocol/openid-connect/token",
		UserInfoURL:  baseURL + "/protocol/openid-connect/userinfo",
		DiscoveryURL: baseURL + "/.well-known/openid_configuration",
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "preferred_username",
			NameField:     "name",
			RoleField:     "realm_access.roles",
			RoleMapping: map[string]string{
				"admin":     "admin",
				"moderator": "moderator",
				"user":      "user",
			},
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// AzureADProvider returns a pre-configured Azure AD OAuth2 provider
func AzureADProvider(clientID, clientSecret, redirectURL, tenantID string) *OAuth2Provider {
	baseURL := "https://login.microsoftonline.com/" + tenantID
	
	return &OAuth2Provider{
		ID:           "azuread",
		Name:         "Azure AD",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      baseURL + "/oauth2/v2.0/authorize",
		TokenURL:     baseURL + "/oauth2/v2.0/token",
		UserInfoURL:  "https://graph.microsoft.com/v1.0/me",
		DiscoveryURL: baseURL + "/v2.0/.well-known/openid_configuration",
		Scopes:       []string{"openid", "profile", "email", "User.Read"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "mail",
			UsernameField: "userPrincipalName",
			NameField:     "displayName",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// GenericOIDCProvider returns a generic OpenID Connect provider
func GenericOIDCProvider(id, name, clientID, clientSecret, redirectURL, discoveryURL string) *OAuth2Provider {
	return &OAuth2Provider{
		ID:           id,
		Name:         name,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		DiscoveryURL: discoveryURL,
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURL:  redirectURL,
		UserMapping: &UserMapping{
			EmailField:    "email",
			UsernameField: "preferred_username",
			NameField:     "name",
		},
		AutoProvision: true,
		DefaultRole:   "user",
		Enabled:       true,
	}
}

// ProviderFactory helps create providers with common patterns
type ProviderFactory struct {
	baseRedirectURL string
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(baseRedirectURL string) *ProviderFactory {
	return &ProviderFactory{
		baseRedirectURL: baseRedirectURL,
	}
}

// CreateGoogle creates a Google provider with the factory's redirect URL
func (pf *ProviderFactory) CreateGoogle(clientID, clientSecret string) *OAuth2Provider {
	return GoogleProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/google")
}

// CreateMicrosoft creates a Microsoft provider with the factory's redirect URL
func (pf *ProviderFactory) CreateMicrosoft(clientID, clientSecret string) *OAuth2Provider {
	return MicrosoftProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/microsoft")
}

// CreateGitHub creates a GitHub provider with the factory's redirect URL
func (pf *ProviderFactory) CreateGitHub(clientID, clientSecret string) *OAuth2Provider {
	return GitHubProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/github")
}

// CreateGitLab creates a GitLab provider with the factory's redirect URL
func (pf *ProviderFactory) CreateGitLab(clientID, clientSecret string, gitlabURL ...string) *OAuth2Provider {
	return GitLabProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/gitlab", gitlabURL...)
}

// CreateOkta creates an Okta provider with the factory's redirect URL
func (pf *ProviderFactory) CreateOkta(clientID, clientSecret, oktaDomain string) *OAuth2Provider {
	return OktaProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/okta", oktaDomain)
}

// CreateAuth0 creates an Auth0 provider with the factory's redirect URL
func (pf *ProviderFactory) CreateAuth0(clientID, clientSecret, auth0Domain string) *OAuth2Provider {
	return Auth0Provider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/auth0", auth0Domain)
}

// CreateKeycloak creates a Keycloak provider with the factory's redirect URL
func (pf *ProviderFactory) CreateKeycloak(clientID, clientSecret, keycloakURL, realm string) *OAuth2Provider {
	return KeycloakProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/keycloak", keycloakURL, realm)
}

// CreateAzureAD creates an Azure AD provider with the factory's redirect URL
func (pf *ProviderFactory) CreateAzureAD(clientID, clientSecret, tenantID string) *OAuth2Provider {
	return AzureADProvider(clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/azuread", tenantID)
}

// CreateGenericOIDC creates a generic OIDC provider with the factory's redirect URL
func (pf *ProviderFactory) CreateGenericOIDC(id, name, clientID, clientSecret, discoveryURL string) *OAuth2Provider {
	return GenericOIDCProvider(id, name, clientID, clientSecret, pf.baseRedirectURL+"/oauth2/callback/"+id, discoveryURL)
}