package build

import "time"

type BuildHandler struct{}

// CreateBuild
type InsertBuildBody struct {
	ProjectId int `json:"project_id"`
}

type RepoAPIResponse struct {
	CommitsURL string `json:"commits_url"`
}

type CommitObject struct {
	Sha string `json:"sha"`
}

// ListBuilds
type ListBuildsBody struct {
	ProjectId int `json:"project_id"`
}

type Build struct {
	Id           int        `json:"build_id"`
	Build_status string     `json:"build_status"`
	Triggered_by string     `json:"triggered_by"`
	Commit_hash  string     `json:"commit_hash"`
	Build_logs   *string    `json:"build_logs,omitempty"`
	Start_time   *time.Time `json:"start_time"`
	End_time     *time.Time `json:"end_time"`
	Created_at   time.Time  `json:"created_at"`
	Updated_at   time.Time  `json:"updated_at"`
}

// Build
type BuildBody struct {
	BuildId int `json:"build_id"`
}
