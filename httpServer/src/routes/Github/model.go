package github

type GithubHandler struct{}

type RepoAPIResponse []struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Owner    struct {
		AvatarURL string `json:"avatar_url"`
	} `json:"owner"`
}
