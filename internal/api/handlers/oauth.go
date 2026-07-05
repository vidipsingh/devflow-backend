package handlers

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
	"strings"

    api "devflow-backend/internal/api/response"
    "devflow-backend/internal/service"

    "github.com/gin-gonic/gin"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/github"
    "golang.org/x/oauth2/google"
)

func githubOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
        ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
        RedirectURL:  os.Getenv("GITHUB_CALLBACK_URL"),
        Scopes:       []string{"user:email", "read:user"},
        Endpoint:     github.Endpoint,
	}
}

func googleOAuthConfig() *oauth2.Config {
    return &oauth2.Config{
        ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        RedirectURL:  os.Getenv("GOOGLE_CALLBACK_URL"),
        Scopes: []string{
            "https://www.googleapis.com/auth/userinfo.email",
            "https://www.googleapis.com/auth/userinfo.profile",
        },
        Endpoint: google.Endpoint,
    }
}

// Github
func GitHubRedirect(c *gin.Context) {
	cfg := githubOAuthConfig()
	authURL := cfg.AuthCodeURL("devflow-state-github", oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func GitHubCallback(c *gin.Context) {
	frontendURL := os.Getenv("FRONTEND_URL")
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/login?error=oauth_cancelled")
		return
	}

	cfg := githubOAuthConfig()

	// Exchange authorization code for access token
	token, err := cfg.Exchange(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL)
		return
	}

	// Fetch user profile from GitHub API
	ghUser, err := fetchGithubUser(token.AccessToken)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL)
		return
	}

	// Fetch primary email
	if ghUser.Email == "" {
		ghUser.Email, _ = fetchGitHubPrimaryEmail(token.AccessToken)
	}

	result, err := service.HandleOAuthUser(c.Request.Context(), service.OAuthUserInfo{
		Provider:  "github",
        OAuthID:   fmt.Sprintf("%d", ghUser.ID),
        Email:     ghUser.Email,
        Name:      ghUser.Name,
        Username:  ghUser.Login,
        AvatarURL: ghUser.AvatarURL,
	})
	if err != nil {
		api.InternalError(c, "OAuth login failed")
		return
	}

	// Redirect to frontend with token in query param
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, url.QueryEscape(result.Token))
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// Google
func GoogleRedirect(c *gin.Context) {
	cfg := googleOAuthConfig()
	authURL := cfg.AuthCodeURL("devflow-state-google", oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func GoogleCallback(c *gin.Context) {
	frontendURL := os.Getenv("FRONTEND_URL")
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/login?error=oauth_cancelled")
		return
	}

	cfg := googleOAuthConfig()

	token, err := cfg.Exchange(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/login?error=oauth_failed")
		return
	}

    // Fetch user profile from Google's userinfo endpoint
	gUser, err := fetchGoogleUser(token.AccessToken)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL)
		return
	}

	// Derive a username from their name or email prefix
	username := strings.Split(gUser.Email, "@")[0]
	if gUser.GivenName != "" {
		username = strings.ToLower(gUser.GivenName)
	}

	result, err := service.HandleOAuthUser(c.Request.Context(), service.OAuthUserInfo{
		Provider:  "google",
        OAuthID:   gUser.ID,
        Email:     gUser.Email,
        Name:      gUser.Name,
        Username:  username,
        AvatarURL: gUser.Picture,
	})
	if err != nil {
		api.InternalError(c, "OAuth login failed")
		return
	}

	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, url.QueryEscape(result.Token))
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

type githubUserResponse struct {
    ID        int    `json:"id"`
    Login     string `json:"login"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    AvatarURL string `json:"avatar_url"`
}

func fetchGithubUser(accessToken string) (*githubUserResponse, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
    req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user githubUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

type githubEmail struct {
    Email    string `json:"email"`
    Primary  bool   `json:"primary"`
    Verified bool   `json:"verified"`
}

func fetchGitHubPrimaryEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
    req.Header.Set("Authorization", "Bearer "+accessToken)
    req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var emails []githubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}

type googleUserResponse struct {
    ID         string `json:"id"`
    Email      string `json:"email"`
    Name       string `json:"name"`
    GivenName  string `json:"given_name"`
    Picture    string `json:"picture"`
}

func fetchGoogleUser(accessToken string) (*googleUserResponse, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user googleUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}