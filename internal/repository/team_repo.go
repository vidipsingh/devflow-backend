package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func teamsCol() *mongo.Collection     { return database.Collection("teams") }
func membersCol() *mongo.Collection   { return database.Collection("team_members") }
func invitesCol() *mongo.Collection   { return database.Collection("team_invites") }
func joinReqsCol() *mongo.Collection  { return database.Collection("team_join_requests") }

// Teams
func CreateTeam(ctx context.Context, team *models.Team) error {
	team.ID = bson.NewObjectID()
	_, err := teamsCol().InsertOne(ctx, team)
	return err
}

func FindTeamByID(ctx context.Context, id bson.ObjectID) (*models.Team, error) {
	var t models.Team
	err := teamsCol().FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &t, err
}

func FindTeamBySlug(ctx context.Context, slug string) (*models.Team, error) {
	var t models.Team
	err := teamsCol().FindOne(ctx, bson.M{"slug": slug}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &t, err
}

// FindTeamsByMember returns all teams that include userID as a member.
func FindTeamsByMember(ctx context.Context, userID bson.ObjectID, limit int64) ([]models.Team, error) {
	// Get teamIDs from memberships first
	cur, err := membersCol().Find(ctx, bson.M{"userId": userID, "isBanned": false},
		options.Find().SetProjection(bson.M{"teamId": 1}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx) 

	var memberships []struct {
		TeamID bson.ObjectID `bson:"teamId"`
	}
	if err := cur.All(ctx, &memberships); err != nil {
		return nil, err
	}

	ids := make([]bson.ObjectID, len(memberships))
	for i, m := range memberships {
		ids[i] = m.TeamID
	}

	if len(ids) == 0 {
		return []models.Team{}, nil
	}

	cur2, err := teamsCol().Find(ctx, bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur2.Close(ctx)
	var teams []models.Team
	return teams, cur2.All(ctx, &teams)
}

// FindPublicTeams lists public teams, optionally filtered by search.
func FindPublicTeams(ctx context.Context, search string, limit, skip int64) ([]models.Team, error) {
	filter := bson.M{"visibility": "public"}
	if search != "" {
		filter["$or"] = bson.A{
			bson.M{"name": bson.M{"$regex": search, "$options": "i"}},
			bson.M{"slug": bson.M{"$regex": search, "$options": "i"}},
			bson.M{"tags": search},
		}
	}
	cur, err := teamsCol().Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "memberCount", Value: -1}}).
			SetLimit(limit).SetSkip(skip))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var teams []models.Team
	return teams, cur.All(ctx, &teams)
}

func FindSubTeams(ctx context.Context, parentID bson.ObjectID) ([]models.Team, error) {
	cur, err := teamsCol().Find(ctx, bson.M{"parentId": parentID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var teams []models.Team
	return teams, cur.All(ctx, &teams)
}

func UpdateTeam(ctx context.Context, id bson.ObjectID, update bson.M) error {
	_, err := teamsCol().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func DeleteTeam(ctx context.Context, id bson.ObjectID) error {
	_, err := teamsCol().DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func IncrTeamStat(ctx context.Context, teamID bson.ObjectID, field string, delta int) error {
	_, err := teamsCol().UpdateOne(ctx, bson.M{"_id": teamID},
		bson.M{"$inc": bson.M{field: delta}})
	return err
}

// Members
func InsertTeamMember(ctx context.Context, m *models.TeamMember) error {
	m.ID = bson.NewObjectID()
	_, err := membersCol().InsertOne(ctx, m)
	return err
}

func FindTeamMember(ctx context.Context, teamID, userID bson.ObjectID) (*models.TeamMember, error) {
	var m models.TeamMember
	err := membersCol().FindOne(ctx, bson.M{"teamId": teamID, "userId": userID}).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &m, err
}

func FindTeamMembers(ctx context.Context, teamID bson.ObjectID, role string, limit, skip int64) ([]models.TeamMember, error) {
	filter := bson.M{"teamId": teamID, "isBanned": false}
	if role != "" {
		filter["role"] = role
	}
	cur, err := membersCol().Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "joinedAt", Value: 1}}).
			SetLimit(limit).SetSkip(skip))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var members []models.TeamMember
	return members, cur.All(ctx, &members)
}

func CountTeamMembers(ctx context.Context, teamID bson.ObjectID) (int64, error) {
	return membersCol().CountDocuments(ctx, bson.M{"teamId": teamID, "isBanned": false})
}

func UpdateTeamMemberRole(ctx context.Context, teamID, userID bson.ObjectID, role string) error {
	_, err := membersCol().UpdateOne(ctx,
		bson.M{"teamId": teamID, "userId": userID},
		bson.M{"$set": bson.M{"role": role, "updatedAt": time.Now()}})
	return err
}

func SetTeamMemberBanned(ctx context.Context, teamID, userID bson.ObjectID, banned bool) error {
	_, err := membersCol().UpdateOne(ctx,
		bson.M{"teamId": teamID, "userId": userID},
		bson.M{"$set": bson.M{"isBanned": banned, "updatedAt": time.Now()}})
	return err
}

func RemoveTeamMember(ctx context.Context, teamID, userID bson.ObjectID) error {
	_, err := membersCol().DeleteOne(ctx, bson.M{"teamId": teamID, "userId": userID})
	return err
}

// Invite
func GenerateInviteToken() (string, error) {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func InsertTeamInvite(ctx context.Context, inv *models.TeamInvite) error {
	inv.ID = bson.NewObjectID()
	_, err := invitesCol().InsertOne(ctx, inv)
	return err
}

func FindInviteByToken(ctx context.Context, token string) (*models.TeamInvite, error) {
	var inv models.TeamInvite
	err := invitesCol().FindOne(ctx, bson.M{"token": token}).Decode(&inv)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &inv, err
}

func FindInviteByTeamAndInvitee(ctx context.Context, teamID bson.ObjectID, inviteeID *bson.ObjectID, email string) (*models.TeamInvite, error) {
	filter := bson.M{"teamId": teamID, "status": "pending"}
	if inviteeID != nil {
		filter["inviteeId"] = *inviteeID
	} else {
		filter["inviteeEmail"] = email
	}
	var inv models.TeamInvite
	err := invitesCol().FindOne(ctx, filter).Decode(&inv)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &inv, err
}

func FindPendingInvitesByTeam(ctx context.Context, teamID bson.ObjectID) ([]models.TeamInvite, error) {
	cur, err := invitesCol().Find(ctx, bson.M{"teamId": teamID, "status": "pending"},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var invs []models.TeamInvite
	return invs, cur.All(ctx, &invs)
}

func FindPendingInvitesByUser(ctx context.Context, userID bson.ObjectID, email string) ([]models.TeamInvite, error) {
	cur, err := invitesCol().Find(ctx,
		bson.M{"$or": bson.A{
			bson.M{"inviteeId": userID, "status": "pending"},
			bson.M{"inviteeEmail": email, "status": "pending"},
		}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var invs []models.TeamInvite
	return invs, cur.All(ctx, &invs)
}

func UpdateInviteStatus(ctx context.Context, id bson.ObjectID, status string) error {
	_, err := invitesCol().UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now()}})
	return err
}

// ExpireOldInvites sets status=expired for all invites past their expiry time.
func ExpireOldInvites(ctx context.Context) error {
	_, err := invitesCol().UpdateMany(ctx,
		bson.M{"status": "pending", "expiresAt": bson.M{"$lt": time.Now()}},
		bson.M{"$set": bson.M{"status": "expired"}})
	return err
}

// Join Requests
func InsertJoinRequest(ctx context.Context, jr *models.TeamJoinRequest) error {
	jr.ID = bson.NewObjectID()
	_, err := joinReqsCol().InsertOne(ctx, jr)
	return err
}

func FindJoinRequest(ctx context.Context, teamID, userID bson.ObjectID) (*models.TeamJoinRequest, error) {
	var jr models.TeamJoinRequest
	err := joinReqsCol().FindOne(ctx, bson.M{"teamId": teamID, "userId": userID, "status": "pending"}).Decode(&jr)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &jr, err
}

func FindJoinRequestByID(ctx context.Context, id bson.ObjectID) (*models.TeamJoinRequest, error) {
	var jr models.TeamJoinRequest
	err := joinReqsCol().FindOne(ctx, bson.M{"_id": id}).Decode(&jr)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &jr, err
}

func FindPendingJoinRequests(ctx context.Context, teamID bson.ObjectID) ([]models.TeamJoinRequest, error) {
	cur, err := joinReqsCol().Find(ctx, bson.M{"teamId": teamID, "status": "pending"},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var reqs []models.TeamJoinRequest
	return reqs, cur.All(ctx, &reqs)
}

func UpdateJoinRequestStatus(ctx context.Context, id bson.ObjectID, status string, reviewerID bson.ObjectID) error {
	now := time.Now()
	_, err := joinReqsCol().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status":     status,
		"reviewedBy": reviewerID,
		"reviewedAt": now,
	}})
	return err
}
