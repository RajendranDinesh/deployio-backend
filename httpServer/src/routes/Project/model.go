package project

import "time"

type ProjectHandler struct{}

type Project struct {
	Name           string  `json:"name"`
	GithubId       string  `json:"github_id"`
	InstallCommand *string `json:"install_command"`
	BuildCommand   *string `json:"build_command"`
	OutputFolder   *string `json:"output_folder"`
	NodeVersion    *string `json:"node_version"`
	Directory      *string `json:"directory"`
}

type ListProject struct {
	Id             int       `json:"id"`
	Name           string    `json:"name"`
	InstallCommand string    `json:"install_command"`
	BuildCommand   string    `json:"build_command"`
	OutputFolder   string    `json:"output_folder"`
	NodeVersion    string    `json:"node_version"`
	Directory      string    `json:"directory"`
	CreatedAt      time.Time `json:"created_at"`
}

type Environment struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type InsertEnvironmentBody struct {
	ProjectId    int           `json:"project_id"`
	Environments []Environment `json:"environments"`
}

type ListEnvKeysBody struct {
	ProjectId int `json:"project_id"`
}

type UpdateEnvironmentBody struct {
	ProjectId int    `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}
