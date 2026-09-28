package workers

import (
    "context"
    "log"

    "devflow-backend/internal/kafka"
    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"
)

// StartNotificationWorker consumes repo/pr/issue events and writes
// per-recipient Notification documents to MongoDB.
func StartNotificationWorker(ctx context.Context) {
	kafka.StartConsumer(ctx, kafka.TopicRepoEvents,  "notif-repo",  handleRepoNotification)
    kafka.StartConsumer(ctx, kafka.TopicPREvents,    "notif-pr",    handlePRNotification)
    kafka.StartConsumer(ctx, kafka.TopicIssueEvents, "notif-issue", handleIssueNotification)
}

// repo events
func handleRepoNotification(ctx context.Context, raw []byte) error {
	var e kafka.RepoEvent
	if err := kafka.DecodeJSON(raw, &e); err != nil {
		return err
	}

	switch e.Type {
	case "repo.forked":
		// Notify the upstream owner that someone forked their repo.
        // OwnerID is the upstream owner; ActorID is the person who forked.
        if e.OwnerID == e.ActorID {
			return nil
		}
		n := &models.Notification{
            RecipientID: e.OwnerID,
            Type:        "repo.forked",
            ActorID:     e.ActorID,
            ActorName:   e.ActorName,
            RepoID:      e.RepoID,
            RepoName:    e.FullName,
            Meta:        map[string]any{"repoSlug": e.RepoSlug},
            Read:        false,
        }
		if err := repository.InsertNotification(ctx, n); err != nil {
			log.Printf("[notif] failed to insert repo.forked notification: %v", err)
		}
	}
	return nil
}

// PR events
func handlePRNotification(ctx context.Context, raw []byte) error {
	var e kafka.PREvent
	if err := kafka.DecodeJSON(raw, &e); err != nil {
		return err
	}

	switch e.Type {
	case "pr.created":
		// Notify the repo owner that a new PR was opened.
        // We need the repo owner — stored as OwnerID on RepoEvent but not on PREvent.
        // Look up the repo to get OwnerID.
        repo, err := repository.FindRepoByID_Hex(ctx, e.RepoID)
		if err != nil || repo == nil {
			return nil
		}
		ownerID := repo.OwnerID.Hex()
		if ownerID == e.ActorID {
			return nil
		}
		n := &models.Notification{
            RecipientID: ownerID,
            Type:        "pr.created",
            ActorID:     e.ActorID,
            ActorName:   e.ActorName,
            RepoID:      e.RepoID,
            RepoName:    e.FullName,
            Meta: map[string]any{
                "prNumber": e.PRNumber,
                "prTitle":  e.PRTitle,
                "repoSlug": e.RepoSlug,
            },
            Read: false,
        }
		if err := repository.InsertNotification(ctx, n); err != nil {
			log.Printf("[notif] failed to insert pr.created notification: %v", err)
		}
	
	case "pr.merged":
		// Notify the PR author that their PR was merged (if someone else merged it).
		if e.AuthorID == e.ActorID {
			return nil
		}
		n := &models.Notification{
            RecipientID: e.AuthorID,
            Type:        "pr.merged",
            ActorID:     e.ActorID,
            ActorName:   e.ActorName,
            RepoID:      e.RepoID,
            RepoName:    e.FullName,
            Meta: map[string]any{
                "prNumber": e.PRNumber,
                "prTitle":  e.PRTitle,
                "repoSlug": e.RepoSlug,
            },
            Read: false,
        }
		if err := repository.InsertNotification(ctx, n); err != nil {
			log.Printf("[notif] failed to insert pr.merged notification: %v", err)
		}
	}
	return nil
}

// Issue Events
func handleIssueNotification(ctx context.Context, raw []byte) error {
	var e kafka.IssueEvent
	if err := kafka.DecodeJSON(raw, &e); err != nil {
		return err
	}

	switch e.Type {
	case "issue.created":
		// Notify the repo owner that a new issue was opened.
		repo, err := repository.FindRepoByID_Hex(ctx, e.RepoID)
		if err != nil || repo == nil {
			return nil
		}
		ownerID := repo.OwnerID.Hex()
		if ownerID == e.ActorID {
			return nil
		}
		n := &models.Notification{
            RecipientID: ownerID,
            Type:        "issue.created",
            ActorID:     e.ActorID,
            ActorName:   e.ActorName,
            RepoID:      e.RepoID,
            RepoName:    e.FullName,
            Meta: map[string]any{
                "issueNumber": e.IssueNumber,
                "issueTitle":  e.IssueTitle,
                "repoSlug":    e.RepoSlug,
            },
            Read: false,
        }
		if err := repository.InsertNotification(ctx, n); err != nil {
			log.Printf("[notif] failed to insert issue.created notification: %v", err)
		}
	
	case "issue.closed":
		// Notify the issue author that their issue was closed (if someone else closed it).
		if e.AuthorID == e.ActorID {
			return nil
		}
		n := &models.Notification{
            RecipientID: e.AuthorID,
            Type:        "issue.closed",
            ActorID:     e.ActorID,
            ActorName:   e.ActorName,
            RepoID:      e.RepoID,
            RepoName:    e.FullName,
            Meta: map[string]any{
                "issueNumber": e.IssueNumber,
                "issueTitle":  e.IssueTitle,
                "repoSlug":    e.RepoSlug,
            },
            Read: false,
        }
		if err := repository.InsertNotification(ctx, n); err != nil {
			log.Printf("[notif] failed to insert issue.closed notification: %v", err)
		}
	}
	return nil
}
