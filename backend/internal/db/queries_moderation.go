package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateBlock(ctx context.Context, actorDID, subjectDID string) error {
	return db.q.CreateBlock(ctx, sqlcdb.CreateBlockParams{
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
		CreatedAt:  time.Now(),
	})
}

func (db *DB) DeleteBlock(ctx context.Context, actorDID, subjectDID string) error {
	return db.q.DeleteBlock(ctx, sqlcdb.DeleteBlockParams{
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
	})
}

func (db *DB) GetBlocks(ctx context.Context, actorDID string) ([]Block, error) {
	rows, err := db.q.GetBlocks(ctx, actorDID)
	if err != nil {
		return nil, err
	}
	var blocks []Block
	for _, r := range rows {
		blocks = append(blocks, Block{
			ID:         int(r.ID),
			ActorDID:   r.ActorDid,
			SubjectDID: r.SubjectDid,
			CreatedAt:  r.CreatedAt,
		})
	}
	return blocks, nil
}

func (db *DB) IsBlocked(ctx context.Context, actorDID, subjectDID string) (bool, error) {
	return db.q.IsBlocked(ctx, sqlcdb.IsBlockedParams{
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
	})
}

func (db *DB) IsBlockedEither(ctx context.Context, did1, did2 string) (bool, error) {
	return db.q.IsBlockedEither(ctx, sqlcdb.IsBlockedEitherParams{
		ActorDid:   did1,
		SubjectDid: did2,
	})
}

func (db *DB) GetBlockedDIDs(ctx context.Context, actorDID string) ([]string, error) {
	return db.q.GetBlockedDIDs(ctx, actorDID)
}

func (db *DB) GetBlockedByDIDs(ctx context.Context, actorDID string) ([]string, error) {
	return db.q.GetBlockedByDIDs(ctx, actorDID)
}

func (db *DB) CreateMute(ctx context.Context, actorDID, subjectDID string) error {
	return db.q.CreateMute(ctx, sqlcdb.CreateMuteParams{
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
		CreatedAt:  time.Now(),
	})
}

func (db *DB) DeleteMute(ctx context.Context, actorDID, subjectDID string) error {
	return db.q.DeleteMute(ctx, sqlcdb.DeleteMuteParams{
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
	})
}

func (db *DB) GetMutes(ctx context.Context, actorDID string) ([]Mute, error) {
	rows, err := db.q.GetMutes(ctx, actorDID)
	if err != nil {
		return nil, err
	}
	var mutes []Mute
	for _, r := range rows {
		mutes = append(mutes, Mute{
			ID:         int(r.ID),
			ActorDID:   r.ActorDid,
			SubjectDID: r.SubjectDid,
			CreatedAt:  r.CreatedAt,
		})
	}
	return mutes, nil
}

func (db *DB) IsMuted(ctx context.Context, actorDID, subjectDID string) (bool, error) {
	return db.q.IsMuted(ctx, sqlcdb.IsMutedParams{
		ActorDid:   actorDID,
		SubjectDid: subjectDID,
	})
}

func (db *DB) GetMutedDIDs(ctx context.Context, actorDID string) ([]string, error) {
	return db.q.GetMutedDIDs(ctx, actorDID)
}

func (db *DB) GetAllHiddenDIDs(ctx context.Context, actorDID string) (map[string]bool, error) {
	hidden := make(map[string]bool)
	if actorDID == "" {
		return hidden, nil
	}

	blocked, err := db.GetBlockedDIDs(ctx, actorDID)
	if err != nil {
		return hidden, err
	}
	for _, did := range blocked {
		hidden[did] = true
	}

	blockedBy, err := db.GetBlockedByDIDs(ctx, actorDID)
	if err != nil {
		return hidden, err
	}
	for _, did := range blockedBy {
		hidden[did] = true
	}

	muted, err := db.GetMutedDIDs(ctx, actorDID)
	if err != nil {
		return hidden, err
	}
	for _, did := range muted {
		hidden[did] = true
	}

	return hidden, nil
}

func (db *DB) GetViewerRelationship(ctx context.Context, viewerDID, subjectDID string) (blocked bool, muted bool, blockedBy bool, err error) {
	if viewerDID == "" || subjectDID == "" {
		return false, false, false, nil
	}

	blocked, err = db.IsBlocked(ctx, viewerDID, subjectDID)
	if err != nil {
		return
	}

	muted, err = db.IsMuted(ctx, viewerDID, subjectDID)
	if err != nil {
		return
	}

	blockedBy, err = db.IsBlocked(ctx, subjectDID, viewerDID)
	return
}

func (db *DB) CreateReport(ctx context.Context, reporterDID, subjectDID string, subjectURI *string, reasonType string, reasonText *string) (int, error) {
	id, err := db.q.CreateReport(ctx, sqlcdb.CreateReportParams{
		ReporterDid: reporterDID,
		SubjectDid:  subjectDID,
		SubjectUri:  subjectURI,
		ReasonType:  reasonType,
		ReasonText:  reasonText,
		CreatedAt:   time.Now(),
	})
	return int(id), err
}

func (db *DB) GetReports(ctx context.Context, status string, limit, offset int) ([]ModerationReport, error) {
	var rows []sqlcdb.ModerationReport
	var err error
	if status != "" {
		rows, err = db.q.GetReportsByStatus(ctx, sqlcdb.GetReportsByStatusParams{
			Status: status,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
	} else {
		rows, err = db.q.GetReports(ctx, sqlcdb.GetReportsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
	}
	if err != nil {
		return nil, err
	}

	var reports []ModerationReport
	for _, r := range rows {
		reports = append(reports, ModerationReport{
			ID:          int(r.ID),
			ReporterDID: r.ReporterDid,
			SubjectDID:  r.SubjectDid,
			SubjectURI:  r.SubjectUri,
			ReasonType:  r.ReasonType,
			ReasonText:  r.ReasonText,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
			ResolvedAt:  r.ResolvedAt,
			ResolvedBy:  r.ResolvedBy,
		})
	}
	return reports, nil
}

func (db *DB) GetReport(ctx context.Context, id int) (*ModerationReport, error) {
	r, err := db.q.GetReport(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	return &ModerationReport{
		ID:          int(r.ID),
		ReporterDID: r.ReporterDid,
		SubjectDID:  r.SubjectDid,
		SubjectURI:  r.SubjectUri,
		ReasonType:  r.ReasonType,
		ReasonText:  r.ReasonText,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		ResolvedAt:  r.ResolvedAt,
		ResolvedBy:  r.ResolvedBy,
	}, nil
}

func (db *DB) ResolveReport(ctx context.Context, id int, resolvedBy string, status string) error {
	now := time.Now()
	return db.q.ResolveReport(ctx, sqlcdb.ResolveReportParams{
		Status:     status,
		ResolvedAt: &now,
		ResolvedBy: &resolvedBy,
		ID:         int32(id),
	})
}

func (db *DB) CreateModerationAction(ctx context.Context, reportID int, actorDID, action string, comment *string) error {
	return db.q.CreateModerationAction(ctx, sqlcdb.CreateModerationActionParams{
		ReportID:  int32(reportID),
		ActorDid:  actorDID,
		Action:    action,
		Comment:   comment,
		CreatedAt: time.Now(),
	})
}

func (db *DB) GetReportActions(ctx context.Context, reportID int) ([]ModerationAction, error) {
	rows, err := db.q.GetReportActions(ctx, int32(reportID))
	if err != nil {
		return nil, err
	}
	var actions []ModerationAction
	for _, r := range rows {
		actions = append(actions, ModerationAction{
			ID:        int(r.ID),
			ReportID:  int(r.ReportID),
			ActorDID:  r.ActorDid,
			Action:    r.Action,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt,
		})
	}
	return actions, nil
}

func (db *DB) GetReportCount(ctx context.Context, status string) (int, error) {
	if status != "" {
		n, err := db.q.GetReportCountByStatus(ctx, status)
		return int(n), err
	}
	n, err := db.q.GetReportCount(ctx)
	return int(n), err
}

func (db *DB) CreateContentLabel(ctx context.Context, src, uri, val, createdBy string) error {
	return db.q.CreateContentLabel(ctx, sqlcdb.CreateContentLabelParams{
		Src:       src,
		Uri:       uri,
		Val:       val,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	})
}

func (db *DB) SyncSelfLabels(ctx context.Context, authorDID, uri string, labels []string) error {
	if err := db.q.DeleteSelfLabels(ctx, sqlcdb.DeleteSelfLabelsParams{
		Src:       authorDID,
		Uri:       uri,
		CreatedBy: authorDID,
	}); err != nil {
		return err
	}
	for _, val := range labels {
		if err := db.CreateContentLabel(ctx, authorDID, uri, val, authorDID); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) NegateContentLabel(ctx context.Context, id int) error {
	return db.q.NegateContentLabel(ctx, int32(id))
}

func (db *DB) DeleteContentLabel(ctx context.Context, id int) error {
	return db.q.DeleteContentLabel(ctx, int32(id))
}

func (db *DB) GetContentLabelsForURIs(ctx context.Context, uris []string, labelerDIDs []string) (map[string][]ContentLabel, error) {
	result := make(map[string][]ContentLabel)
	if len(uris) == 0 {
		return result, nil
	}

	var rows []sqlcdb.ContentLabel
	var err error
	if len(labelerDIDs) > 0 {
		rows, err = db.q.GetContentLabelsForURIsBySrc(ctx, sqlcdb.GetContentLabelsForURIsBySrcParams{
			Column1: uris,
			Column2: labelerDIDs,
		})
	} else {
		rows, err = db.q.GetContentLabelsForURIs(ctx, uris)
	}
	if err != nil {
		return result, err
	}

	for _, r := range rows {
		l := ContentLabel{
			ID:        int(r.ID),
			Src:       r.Src,
			URI:       r.Uri,
			Val:       r.Val,
			Neg:       r.Neg != 0,
			CreatedBy: r.CreatedBy,
			CreatedAt: r.CreatedAt,
		}
		result[l.URI] = append(result[l.URI], l)
	}
	return result, nil
}

func (db *DB) GetContentLabelsForDIDs(ctx context.Context, dids []string, labelerDIDs []string) (map[string][]ContentLabel, error) {
	result := make(map[string][]ContentLabel)
	if len(dids) == 0 {
		return result, nil
	}

	var rows []sqlcdb.ContentLabel
	var err error
	if len(labelerDIDs) > 0 {
		rows, err = db.q.GetContentLabelsForURIsBySrc(ctx, sqlcdb.GetContentLabelsForURIsBySrcParams{
			Column1: dids,
			Column2: labelerDIDs,
		})
	} else {
		rows, err = db.q.GetContentLabelsForURIs(ctx, dids)
	}
	if err != nil {
		return result, err
	}

	for _, r := range rows {
		l := ContentLabel{
			ID:        int(r.ID),
			Src:       r.Src,
			URI:       r.Uri,
			Val:       r.Val,
			Neg:       r.Neg != 0,
			CreatedBy: r.CreatedBy,
			CreatedAt: r.CreatedAt,
		}
		result[l.URI] = append(result[l.URI], l)
	}
	return result, nil
}

func (db *DB) GetAllContentLabels(ctx context.Context, limit, offset int) ([]ContentLabel, error) {
	rows, err := db.q.GetAllContentLabels(ctx, sqlcdb.GetAllContentLabelsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	var labels []ContentLabel
	for _, r := range rows {
		labels = append(labels, ContentLabel{
			ID:        int(r.ID),
			Src:       r.Src,
			URI:       r.Uri,
			Val:       r.Val,
			Neg:       r.Neg != 0,
			CreatedBy: r.CreatedBy,
			CreatedAt: r.CreatedAt,
		})
	}
	return labels, nil
}

func (db *DB) MarkTakenDown(ctx context.Context, uri string) error {
	return db.q.MarkTakenDown(ctx, sqlcdb.MarkTakenDownParams{
		Uri:         uri,
		TakenDownAt: time.Now(),
	})
}

func (db *DB) IsTakenDown(ctx context.Context, uri string) (bool, error) {
	return db.q.IsTakenDown(ctx, uri)
}

type BannedAccount struct {
	DID      string    `json:"did"`
	Reason   *string   `json:"reason,omitempty"`
	BannedBy string    `json:"bannedBy"`
	BannedAt time.Time `json:"bannedAt"`
}

func (db *DB) BanAccount(ctx context.Context, did, bannedBy string, reason *string) error {
	return db.q.BanAccount(ctx, sqlcdb.BanAccountParams{
		Did:      did,
		Reason:   reason,
		BannedBy: bannedBy,
		BannedAt: time.Now(),
	})
}

func (db *DB) UnbanAccount(ctx context.Context, did string) error {
	return db.q.UnbanAccount(ctx, did)
}

func (db *DB) IsBanned(ctx context.Context, did string) (bool, error) {
	return db.q.IsBanned(ctx, did)
}

func (db *DB) GetBannedAccounts(ctx context.Context) ([]BannedAccount, error) {
	rows, err := db.q.GetBannedAccounts(ctx)
	if err != nil {
		return nil, err
	}
	var accounts []BannedAccount
	for _, r := range rows {
		accounts = append(accounts, BannedAccount{
			DID:      r.Did,
			Reason:   r.Reason,
			BannedBy: r.BannedBy,
			BannedAt: r.BannedAt,
		})
	}
	return accounts, nil
}

func (db *DB) GetBannedDIDs(ctx context.Context) ([]string, error) {
	return db.q.GetBannedDIDs(ctx)
}
