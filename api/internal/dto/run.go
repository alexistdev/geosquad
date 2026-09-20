package dto

import (
	"strings"

	"github.com/alexistdev/geosquad/api/internal/model"
)

type CreateRunRequest struct {
	Request string `json:"request"`
}

func (r *CreateRunRequest) Normalize() {
	r.Request = strings.TrimSpace(r.Request)
}

func (r CreateRunRequest) Validate() []string {
	var errs []string
	switch {
	case r.Request == "":
		errs = append(errs, "Request wajib diisi.")
	case len(r.Request) < 10:
		errs = append(errs, "Request terlalu pendek, jelaskan minimal dalam satu kalimat.")
	case len(r.Request) > 5000:
		errs = append(errs, "Request maksimal 5000 karakter.")
	}
	return errs
}

type RunResponse struct {
	ID           string  `json:"id"`
	Ticket       string  `json:"ticket"`
	UserID       string  `json:"userId"`
	Request      string  `json:"request"`
	Status       string  `json:"status"`
	Branch       *string `json:"branch,omitempty"`
	ExitCode     *int    `json:"exitCode,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	StartedAt    *string `json:"startedAt,omitempty"`
	FinishedAt   *string `json:"finishedAt,omitempty"`
	CreatedDate  string  `json:"createdDate"`
}

func NewRunResponse(r *model.Run) RunResponse {
	return RunResponse{
		ID:           r.ID,
		Ticket:       r.Ticket,
		UserID:       r.UserID,
		Request:      r.Request,
		Status:       string(r.Status),
		Branch:       r.Branch,
		ExitCode:     r.ExitCode,
		ErrorMessage: r.ErrorMessage,
		StartedAt:    formatTimePtr(r.StartedAt),
		FinishedAt:   formatTimePtr(r.FinishedAt),
		CreatedDate:  r.CreatedDate.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func NewRunResponses(runs []model.Run) []RunResponse {
	out := make([]RunResponse, 0, len(runs))
	for i := range runs {
		out = append(out, NewRunResponse(&runs[i]))
	}
	return out
}

type RunLogResponse struct {
	Seq      int    `json:"seq"`
	Stream   string `json:"stream"`
	Line     string `json:"line"`
	LoggedAt string `json:"loggedAt"`
}

func NewRunLogResponses(logs []model.RunLog) []RunLogResponse {
	out := make([]RunLogResponse, 0, len(logs))
	for _, l := range logs {
		out = append(out, RunLogResponse{
			Seq:      l.Seq,
			Stream:   string(l.Stream),
			Line:     l.Line,
			LoggedAt: l.LoggedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		})
	}
	return out
}
