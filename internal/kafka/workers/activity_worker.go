package workers

import (
    "context"
    "log"

    "devflow-backend/internal/kafka"
    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"
)

// StartActivityWorker consumes repo + pr + issue events and writes
// ActivityFeed documents to MongoDB for the dashboard activity feed.
func StartActivityWorker(ctx context.Context) {
    kafka.StartConsumer(ctx, kafka.TopicRepoEvents, "activity-repo", handleRepoActivity)
    kafka.StartConsumer(ctx, kafka.TopicPREvents,   "activity-pr",   handlePRActivity)
    kafka.StartConsumer(ctx, kafka.TopicIssueEvents,"activity-issue",handleIssueActivity)
}

func handleRepoActivity(ctx context.Context, raw []byte) error {
    var e kafka.RepoEvent
    if err := kafka.DecodeJSON(raw, &e); err != nil {
        return err
    }
    log.Printf("[activity] %s by %s on %s", e.Type, e.ActorName, e.FullName)
    return repository.InsertActivity(ctx, &models.ActivityEvent{
        Type:      e.Type,
        ActorID:   e.ActorID,
        ActorName: e.ActorName,
        RepoID:    e.RepoID,
        RepoName:  e.FullName,
        Timestamp: e.Timestamp,
    })
}

func handlePRActivity(ctx context.Context, raw []byte) error {
    var e kafka.PREvent
    if err := kafka.DecodeJSON(raw, &e); err != nil {
        return err
    }
    log.Printf("[activity] %s #%d by %s on %s", e.Type, e.PRNumber, e.ActorName, e.FullName)
    return repository.InsertActivity(ctx, &models.ActivityEvent{
        Type:      e.Type,
        ActorID:   e.ActorID,
        ActorName: e.ActorName,
        RepoID:    e.RepoID,
        RepoName:  e.FullName,
        Meta:      map[string]any{"prNumber": e.PRNumber, "title": e.PRTitle},
        Timestamp: e.Timestamp,
    })
}

func handleIssueActivity(ctx context.Context, raw []byte) error {
    var e kafka.IssueEvent
    if err := kafka.DecodeJSON(raw, &e); err != nil {
        return err
    }
    log.Printf("[activity] %s #%d by %s on %s", e.Type, e.IssueNumber, e.ActorName, e.FullName)
    return repository.InsertActivity(ctx, &models.ActivityEvent{
        Type:      e.Type,
        ActorID:   e.ActorID,
        ActorName: e.ActorName,
        RepoID:    e.RepoID,
        RepoName:  e.FullName,
        Meta:      map[string]any{"issueNumber": e.IssueNumber, "title": e.IssueTitle},
        Timestamp: e.Timestamp,
    })
}
