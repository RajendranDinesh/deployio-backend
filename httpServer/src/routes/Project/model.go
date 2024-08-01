package project

import "time"

type ProjectHandler struct{}

type Project struct {
	Name         string  `json:"name"`
	GithubId     int     `json:"github_id"`
	BuildCommand *string `json:"build_command"`
	OutputFolder *string `json:"output_folder"`
}

type ListProject struct {
	Id           int       `json:"id"`
	Name         string    `json:"name"`
	GithubId     int       `json:"github_id"`
	BuildCommand string    `json:"build_command"`
	OutputFolder string    `json:"output_folder"`
	CreatedAt    time.Time `json:"created_at"`
}

type Environment struct {
	ProjectId int    `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type InsertEnvironmentBody struct {
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
