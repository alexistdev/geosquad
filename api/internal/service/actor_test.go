package service

import (
	"testing"

	"github.com/alexistdev/geosquad/api/internal/model"
)

func TestActorCanAccess(t *testing.T) {
	admin := &Actor{UserID: "admin-1", Role: model.RoleAdmin}
	user := &Actor{UserID: "user-1", Role: model.RoleUser}

	cases := []struct {
		name  string
		actor *Actor
		owner string
		want  bool
	}{
		{"admin ke data sendiri", admin, "admin-1", true},
		{"admin ke data orang lain", admin, "user-1", true},
		{"user ke data sendiri", user, "user-1", true},
		{"user ke data orang lain", user, "user-2", false},
		{"actor nil", nil, "user-1", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.actor.CanAccess(c.owner); got != c.want {
				t.Errorf("CanAccess = %v, mau %v", got, c.want)
			}
		})
	}
}

func TestRunStatusTerminal(t *testing.T) {
	terminal := []model.RunStatus{model.RunPassed, model.RunFailed, model.RunCancelled, model.RunError}
	for _, s := range terminal {
		if !s.Terminal() {
			t.Errorf("%s seharusnya terminal", s)
		}
	}
	for _, s := range []model.RunStatus{model.RunPending, model.RunRunning} {
		if s.Terminal() {
			t.Errorf("%s seharusnya belum terminal", s)
		}
	}
}
