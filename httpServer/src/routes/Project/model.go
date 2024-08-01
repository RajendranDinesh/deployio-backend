package project

type ProjectHandler struct{}

type Project struct {
	Name         string  `json:"name"`
	GithubId     int     `json:"github_id"`
	BuildCommand *string `json:"build_command"`
	OutputFolder *string `json:"output_folder"`
}

type RepositoriesAPIResponse struct {
	CloneURL string `json:"clone_url"`
}
