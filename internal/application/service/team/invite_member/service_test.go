package invite_member_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meindokuse/task-service/internal/application/service/team/invite_member"
	"github.com/meindokuse/task-service/internal/domain/entity"
	"github.com/meindokuse/task-service/internal/domain/valueobject"
	"github.com/meindokuse/task-service/internal/pkg/terror"
)

type mockTeamRepo struct {
	getByIDFn func(ctx context.Context, id uint64) (*entity.Team, error)
}

func (m *mockTeamRepo) GetByID(ctx context.Context, id uint64) (*entity.Team, error) {
	return m.getByIDFn(ctx, id)
}

type mockMemberRepo struct {
	getRoleFn func(ctx context.Context, teamID, userID uint64) (valueobject.Role, error)
	addFn     func(ctx context.Context, teamID, userID uint64, role valueobject.Role) error
}

func (m *mockMemberRepo) GetRole(ctx context.Context, teamID, userID uint64) (valueobject.Role, error) {
	return m.getRoleFn(ctx, teamID, userID)
}

func (m *mockMemberRepo) Add(ctx context.Context, teamID, userID uint64, role valueobject.Role) error {
	return m.addFn(ctx, teamID, userID, role)
}

type mockUserRepo struct {
	getByEmailFn func(ctx context.Context, email string) (*entity.User, error)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return m.getByEmailFn(ctx, email)
}

type mockEmailGateway struct {
	sendInviteFn func(ctx context.Context, toEmail, teamName string) error
}

func (m *mockEmailGateway) SendInvite(ctx context.Context, toEmail, teamName string) error {
	return m.sendInviteFn(ctx, toEmail, teamName)
}

func newHappyPathMocks() (*mockTeamRepo, *mockMemberRepo, *mockUserRepo) {
	teams := &mockTeamRepo{getByIDFn: func(_ context.Context, id uint64) (*entity.Team, error) {
		return &entity.Team{ID: id, Name: "Backend"}, nil
	}}
	members := &mockMemberRepo{
		getRoleFn: func(context.Context, uint64, uint64) (valueobject.Role, error) {
			return valueobject.RoleOwner, nil
		},
		addFn: func(context.Context, uint64, uint64, valueobject.Role) error { return nil },
	}
	users := &mockUserRepo{getByEmailFn: func(_ context.Context, email string) (*entity.User, error) {
		return &entity.User{ID: 10, Email: email}, nil
	}}
	return teams, members, users
}

func TestService_Execute_ForbiddenWhenNotMember(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	members.getRoleFn = func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return "", terror.NotFound("not a team member")
	}
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		t.Fatal("should not send email")
		return nil
	}}

	svc := invite_member.New(teams, members, users, email)
	_, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com"})

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_ForbiddenWhenPlainMember(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	members.getRoleFn = func(context.Context, uint64, uint64) (valueobject.Role, error) {
		return valueobject.RoleMember, nil
	}
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		t.Fatal("should not send email")
		return nil
	}}

	svc := invite_member.New(teams, members, users, email)
	_, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com"})

	require.Error(t, err)
	assert.Equal(t, 403, terror.HTTPStatus(err))
}

func TestService_Execute_InviteeNotFound(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	users.getByEmailFn = func(context.Context, string) (*entity.User, error) {
		return nil, terror.NotFound("user not found")
	}
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		t.Fatal("should not send email")
		return nil
	}}

	svc := invite_member.New(teams, members, users, email)
	_, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "nobody@example.com"})

	require.Error(t, err)
	assert.Equal(t, 404, terror.HTTPStatus(err))
}

func TestService_Execute_SuccessWithEmailDelivered(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error { return nil }}

	svc := invite_member.New(teams, members, users, email)
	res, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com"})

	require.NoError(t, err)
	assert.Equal(t, uint64(10), res.UserID)
	assert.True(t, res.EmailDelivered)
}

func TestService_Execute_TeamLookupErrorPropagates(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	teams.getByIDFn = func(context.Context, uint64) (*entity.Team, error) {
		return nil, terror.Internal("db down", assert.AnError)
	}
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		t.Fatal("should not send email")
		return nil
	}}

	svc := invite_member.New(teams, members, users, email)
	_, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com"})

	require.Error(t, err)
	assert.Equal(t, 500, terror.HTTPStatus(err))
}

func TestService_Execute_AlreadyMemberConflictPropagates(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	members.addFn = func(context.Context, uint64, uint64, valueobject.Role) error {
		return terror.Conflict("user is already a team member")
	}
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		t.Fatal("should not send email when membership add fails")
		return nil
	}}

	svc := invite_member.New(teams, members, users, email)
	_, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com"})

	require.Error(t, err)
	assert.Equal(t, 409, terror.HTTPStatus(err))
}

func TestService_Execute_InvalidRoleRejected(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		t.Fatal("should not send email")
		return nil
	}}

	svc := invite_member.New(teams, members, users, email)
	_, err := svc.Execute(context.Background(), invite_member.Request{
		TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com", Role: valueobject.Role("bogus"),
	})

	require.Error(t, err)
	assert.Equal(t, 400, terror.HTTPStatus(err))
}

func TestService_Execute_SuccessWithEmailFailureDoesNotFailInvite(t *testing.T) {
	teams, members, users := newHappyPathMocks()
	email := &mockEmailGateway{sendInviteFn: func(context.Context, string, string) error {
		return assert.AnError
	}}

	svc := invite_member.New(teams, members, users, email)
	res, err := svc.Execute(context.Background(), invite_member.Request{TeamID: 1, InviterID: 2, InviteeEmail: "a@b.com"})

	require.NoError(t, err)
	assert.Equal(t, uint64(10), res.UserID)
	assert.False(t, res.EmailDelivered)
}
