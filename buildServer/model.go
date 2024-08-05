package main

type Request struct {
	BuildId int `json:"build_id"`
}

type RepoResponse struct {
	ArchiveURL string `json:"archive_url"`
}
