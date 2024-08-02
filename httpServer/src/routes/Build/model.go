package build

type BuildHandler struct{}

type InsertBuildBody struct {
	ProjectId int `json:"project_id"`
}

type RepoAPIResponse struct {
	CommitsURL string `json:"commits_url"`
}
