package team

import (
	"github.com/meindokuse/task-service/internal/application/service/team/create_team"
	"github.com/meindokuse/task-service/internal/application/service/team/invite_member"
	"github.com/meindokuse/task-service/internal/application/service/team/list_teams"
	"github.com/meindokuse/task-service/internal/application/service/team/orphaned_assignees"
	"github.com/meindokuse/task-service/internal/application/service/team/team_stats"
	"github.com/meindokuse/task-service/internal/application/service/team/top_creators"
)

// Registry объединяет все use case'ы команд, создаётся один раз в internal/init.go.
type Registry struct {
	CreateTeam        *create_team.Service
	ListTeams         *list_teams.Service
	InviteMember      *invite_member.Service
	TeamStats         *team_stats.Service
	TopCreators       *top_creators.Service
	OrphanedAssignees *orphaned_assignees.Service
}

func NewRegistry(
	createTeam *create_team.Service,
	listTeams *list_teams.Service,
	inviteMember *invite_member.Service,
	teamStats *team_stats.Service,
	topCreators *top_creators.Service,
	orphanedAssignees *orphaned_assignees.Service,
) *Registry {
	return &Registry{
		CreateTeam:        createTeam,
		ListTeams:         listTeams,
		InviteMember:      inviteMember,
		TeamStats:         teamStats,
		TopCreators:       topCreators,
		OrphanedAssignees: orphanedAssignees,
	}
}
