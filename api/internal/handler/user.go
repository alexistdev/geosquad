package handler

import (
	"net/http"
	"strings"

	"github.com/alexistdev/geosquad/api/internal/dto"
	"github.com/alexistdev/geosquad/api/internal/repository"
	"github.com/alexistdev/geosquad/api/internal/response"
	"github.com/alexistdev/geosquad/api/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	p := parsePagination(r)
	filter := repository.UserFilter{
		Search: strings.TrimSpace(r.URL.Query().Get("search")),
		Role:   strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("role"))),
		Limit:  p.PerPage,
		Offset: p.Offset(),
	}

	users, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		response.FromError(w, err)
		return
	}

	items := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		items = append(items, dto.NewUserResponse(&users[i]))
	}
	response.OK(w, "", response.Paged{Items: items, Meta: meta(p, total)})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "", dto.NewUserResponse(user))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	var req dto.CreateUserRequest
	if !decode(w, r, &req) {
		return
	}
	req.Normalize()
	if errs := req.Validate(); len(errs) > 0 {
		response.BadRequest(w, errs...)
		return
	}

	user, err := h.svc.Create(r.Context(), req, a.Email)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.Created(w, "User berhasil dibuat.", dto.NewUserResponse(user))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}

	var req dto.UpdateUserRequest
	if !decode(w, r, &req) {
		return
	}
	req.Normalize()
	if errs := req.Validate(); len(errs) > 0 {
		response.BadRequest(w, errs...)
		return
	}

	user, err := h.svc.Update(r.Context(), r.PathValue("id"), req, a)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "User berhasil diperbarui.", dto.NewUserResponse(user))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	a, ok := actor(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), a); err != nil {
		response.FromError(w, err)
		return
	}
	response.NoContent(w, "User berhasil dihapus.")
}
