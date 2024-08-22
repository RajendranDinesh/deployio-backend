package deployment

import "time"

type DeploymentHandler struct{}

type Deployment struct {
	Id        int       `json:"id"`
	BuildId   int       `json:"build_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ListDeploymentsReqBody struct {
	ProjectId int `json:"project_id"`
}

type DeleteDeploymentReqBody struct {
	ProjectId int `json:"project_id"`
}
