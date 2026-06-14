-- name: CreateBlock :exec
INSERT INTO blocks (actor_did, subject_did, created_at) VALUES ($1, $2, $3)
ON CONFLICT(actor_did, subject_did) DO NOTHING;

-- name: DeleteBlock :exec
DELETE FROM blocks WHERE actor_did = $1 AND subject_did = $2;

-- name: GetBlocks :many
SELECT id, actor_did, subject_did, created_at FROM blocks WHERE actor_did = $1 ORDER BY created_at DESC;

-- name: IsBlocked :one
SELECT EXISTS(SELECT 1 FROM blocks WHERE actor_did = $1 AND subject_did = $2);

-- name: IsBlockedEither :one
SELECT EXISTS(SELECT 1 FROM blocks WHERE (actor_did = $1 AND subject_did = $2) OR (actor_did = $2 AND subject_did = $1));

-- name: GetBlockedDIDs :many
SELECT subject_did FROM blocks WHERE actor_did = $1;

-- name: GetBlockedByDIDs :many
SELECT actor_did FROM blocks WHERE subject_did = $1;

-- name: CreateMute :exec
INSERT INTO mutes (actor_did, subject_did, created_at) VALUES ($1, $2, $3)
ON CONFLICT(actor_did, subject_did) DO NOTHING;

-- name: DeleteMute :exec
DELETE FROM mutes WHERE actor_did = $1 AND subject_did = $2;

-- name: GetMutes :many
SELECT id, actor_did, subject_did, created_at FROM mutes WHERE actor_did = $1 ORDER BY created_at DESC;

-- name: IsMuted :one
SELECT EXISTS(SELECT 1 FROM mutes WHERE actor_did = $1 AND subject_did = $2);

-- name: GetMutedDIDs :many
SELECT subject_did FROM mutes WHERE actor_did = $1;

-- name: CreateReport :one
INSERT INTO moderation_reports (reporter_did, subject_did, subject_uri, reason_type, reason_text, status, created_at)
VALUES ($1, $2, $3, $4, $5, 'pending', $6)
RETURNING id;

-- name: GetReports :many
SELECT id, reporter_did, subject_did, subject_uri, reason_type, reason_text, status, created_at, resolved_at, resolved_by
FROM moderation_reports
ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetReportsByStatus :many
SELECT id, reporter_did, subject_did, subject_uri, reason_type, reason_text, status, created_at, resolved_at, resolved_by
FROM moderation_reports
WHERE status = $1
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetReport :one
SELECT id, reporter_did, subject_did, subject_uri, reason_type, reason_text, status, created_at, resolved_at, resolved_by FROM moderation_reports WHERE id = $1;

-- name: ResolveReport :exec
UPDATE moderation_reports SET status = $1, resolved_at = $2, resolved_by = $3 WHERE id = $4;

-- name: CreateModerationAction :exec
INSERT INTO moderation_actions (report_id, actor_did, action, comment, created_at) VALUES ($1, $2, $3, $4, $5);

-- name: GetReportActions :many
SELECT id, report_id, actor_did, action, comment, created_at FROM moderation_actions WHERE report_id = $1 ORDER BY created_at DESC;

-- name: GetReportCount :one
SELECT COUNT(*) FROM moderation_reports;

-- name: GetReportCountByStatus :one
SELECT COUNT(*) FROM moderation_reports WHERE status = $1;

-- name: CreateContentLabel :exec
INSERT INTO content_labels (src, uri, val, neg, created_by, created_at) VALUES ($1, $2, $3, 0, $4, $5);

-- name: DeleteSelfLabels :exec
DELETE FROM content_labels WHERE src = $1 AND uri = $2 AND created_by = $3;

-- name: NegateContentLabel :exec
UPDATE content_labels SET neg = 1 WHERE id = $1;

-- name: DeleteContentLabel :exec
DELETE FROM content_labels WHERE id = $1;

-- name: GetContentLabelsForURIs :many
SELECT id, src, uri, val, neg, created_by, created_at FROM content_labels
WHERE uri = ANY($1::text[]) AND neg = 0
ORDER BY created_at DESC;

-- name: GetContentLabelsForURIsBySrc :many
SELECT id, src, uri, val, neg, created_by, created_at FROM content_labels
WHERE uri = ANY($1::text[]) AND neg = 0 AND src = ANY($2::text[])
ORDER BY created_at DESC;

-- name: GetAllContentLabels :many
SELECT id, src, uri, val, neg, created_by, created_at FROM content_labels ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: MarkTakenDown :exec
INSERT INTO taken_down_uris (uri, taken_down_at) VALUES ($1, $2)
ON CONFLICT(uri) DO NOTHING;

-- name: IsTakenDown :one
SELECT EXISTS(SELECT 1 FROM taken_down_uris WHERE uri = $1);

-- name: BanAccount :exec
INSERT INTO banned_accounts (did, reason, banned_by, banned_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT(did) DO UPDATE SET reason = EXCLUDED.reason, banned_by = EXCLUDED.banned_by, banned_at = EXCLUDED.banned_at;

-- name: UnbanAccount :exec
DELETE FROM banned_accounts WHERE did = $1;

-- name: IsBanned :one
SELECT EXISTS(SELECT 1 FROM banned_accounts WHERE did = $1);

-- name: GetBannedAccounts :many
SELECT did, reason, banned_by, banned_at FROM banned_accounts ORDER BY banned_at DESC;

-- name: GetBannedDIDs :many
SELECT did FROM banned_accounts;
