// Package invite_member реализует приглашения в команду: приглашать может
// только owner/admin, а отправка письма идёт через шлюз с circuit breaker,
// чтобы нестабильный почтовый провайдер не приводил к отказу приглашения,
// а деградировал плавно.
package invite_member

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type TeamRepository interface {
	GetByID(ctx context.Context, id uint64) (*entity.Team, error)
}

type TeamMemberRepository interface {
	GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error)
	Add(ctx context.Context, teamID, userID uint64, role valueobject.Role) error
}

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
}

type EmailGateway interface {
	SendInvite(ctx context.Context, toEmail, teamName string) error
}

type Service struct {
	teams   TeamRepository
	members TeamMemberRepository
	users   UserRepository
	email   EmailGateway
}

func New(teams TeamRepository, members TeamMemberRepository, users UserRepository, email EmailGateway) *Service {
	return &Service{teams: teams, members: members, users: users, email: email}
}

type Request struct {
	TeamID       uint64
	InviterID    uint64
	InviteeEmail string
	Role         valueobject.Role
}

type Response struct {
	UserID         uint64
	EmailDelivered bool
}

func (s *Service) Execute(ctx context.Context, req Request) (Response, error) {
	inviterRole, err := s.members.GetRole(ctx, req.TeamID, req.InviterID)
	if err != nil {
		var terr *terror.Error
		if errors.As(err, &terr) && terr.Kind == terror.KindNotFound {
			return Response{}, terror.Forbidden("you are not a member of this team")
		}
		return Response{}, err
	}
	if !inviterRole.CanManageMembers() {
		return Response{}, terror.Forbidden("only team owner/admin can invite members")
	}

	role := req.Role
	if role == "" {
		role = valueobject.RoleMember
	}
	if !role.Valid() {
		return Response{}, terror.Validation("invalid role")
	}

	team, err := s.teams.GetByID(ctx, req.TeamID)
	if err != nil {
		return Response{}, err
	}

	email := strings.TrimSpace(strings.ToLower(req.InviteeEmail))
	invitee, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		var terr *terror.Error
		if errors.As(err, &terr) && terr.Kind == terror.KindNotFound {
			return Response{}, terror.NotFound("no user registered with this email")
		}
		return Response{}, err
	}

	if err := s.members.Add(ctx, req.TeamID, invitee.ID, role); err != nil {
		return Response{}, err
	}

	delivered := true
	if err := s.email.SendInvite(ctx, invitee.Email, team.Name); err != nil {
		delivered = false
		slog.Warn("invite email not delivered", "team_id", req.TeamID, "invitee_id", invitee.ID, "error", err)
	}

	return Response{UserID: invitee.ID, EmailDelivered: delivered}, nil
}
