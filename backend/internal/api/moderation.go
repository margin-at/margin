package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"margin.at/internal/config"
	"margin.at/internal/db"
	"margin.at/internal/logger"
)

type ModerationHandler struct {
	db        *db.DB
	refresher *TokenRefresher
}

func NewModerationHandler(database *db.DB, refresher *TokenRefresher) *ModerationHandler {
	return &ModerationHandler{db: database, refresher: refresher}
}

func (m *ModerationHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DID == "" {
		WriteBadRequest(w, "did is required")
		return
	}

	if req.DID == session.DID {
		WriteBadRequest(w, "Cannot block yourself")
		return
	}

	if err := m.db.CreateBlock(r.Context(), session.DID, req.DID); err != nil {
		logger.Error("Failed to create block: %v", err)
		WriteInternalError(w, "Failed to block user")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	did := r.URL.Query().Get("did")
	if did == "" {
		WriteBadRequest(w, "did query parameter required")
		return
	}

	if err := m.db.DeleteBlock(r.Context(), session.DID, did); err != nil {
		logger.Error("Failed to delete block: %v", err)
		WriteInternalError(w, "Failed to unblock user")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) GetBlocks(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	blocks, err := m.db.GetBlocks(r.Context(), session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to fetch blocks")
		return
	}

	dids := make([]string, len(blocks))
	for i, b := range blocks {
		dids[i] = b.SubjectDID
	}
	profiles := fetchProfilesForDIDs(m.db, dids)

	type BlockedUser struct {
		DID       string `json:"did"`
		Author    Author `json:"author"`
		CreatedAt string `json:"createdAt"`
	}

	items := make([]BlockedUser, len(blocks))
	for i, b := range blocks {
		items[i] = BlockedUser{
			DID:       b.SubjectDID,
			Author:    profiles[b.SubjectDID],
			CreatedAt: b.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	WriteSuccess(w, map[string]interface{}{"items": items})
}

func (m *ModerationHandler) MuteUser(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DID == "" {
		WriteBadRequest(w, "did is required")
		return
	}

	if req.DID == session.DID {
		WriteBadRequest(w, "Cannot mute yourself")
		return
	}

	if err := m.db.CreateMute(r.Context(), session.DID, req.DID); err != nil {
		logger.Error("Failed to create mute: %v", err)
		WriteInternalError(w, "Failed to mute user")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) UnmuteUser(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	did := r.URL.Query().Get("did")
	if did == "" {
		WriteBadRequest(w, "did query parameter required")
		return
	}

	if err := m.db.DeleteMute(r.Context(), session.DID, did); err != nil {
		logger.Error("Failed to delete mute: %v", err)
		WriteInternalError(w, "Failed to unmute user")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) GetMutes(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	mutes, err := m.db.GetMutes(r.Context(), session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to fetch mutes")
		return
	}

	dids := make([]string, len(mutes))
	for i, mu := range mutes {
		dids[i] = mu.SubjectDID
	}
	profiles := fetchProfilesForDIDs(m.db, dids)

	type MutedUser struct {
		DID       string `json:"did"`
		Author    Author `json:"author"`
		CreatedAt string `json:"createdAt"`
	}

	items := make([]MutedUser, len(mutes))
	for i, mu := range mutes {
		items[i] = MutedUser{
			DID:       mu.SubjectDID,
			Author:    profiles[mu.SubjectDID],
			CreatedAt: mu.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	WriteSuccess(w, map[string]interface{}{"items": items})
}

func (m *ModerationHandler) GetRelationship(w http.ResponseWriter, r *http.Request) {
	viewerDID := m.getViewerDID(r)
	subjectDID := r.URL.Query().Get("did")

	if subjectDID == "" {
		WriteBadRequest(w, "did query parameter required")
		return
	}

	blocked, muted, blockedBy, err := m.db.GetViewerRelationship(r.Context(), viewerDID, subjectDID)
	if err != nil {
		WriteInternalError(w, "Failed to get relationship")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"blocking":  blocked,
		"muting":    muted,
		"blockedBy": blockedBy,
	})
}

func (m *ModerationHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req struct {
		SubjectDID string  `json:"subjectDid"`
		SubjectURI *string `json:"subjectUri,omitempty"`
		ReasonType string  `json:"reasonType"`
		ReasonText *string `json:"reasonText,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.SubjectDID == "" || req.ReasonType == "" {
		WriteBadRequest(w, "subjectDid and reasonType are required")
		return
	}

	validReasons := map[string]bool{
		"spam":       true,
		"violation":  true,
		"misleading": true,
		"sexual":     true,
		"rude":       true,
		"other":      true,
	}

	if !validReasons[req.ReasonType] {
		WriteBadRequest(w, "Invalid reasonType")
		return
	}

	id, err := m.db.CreateReport(r.Context(), session.DID, req.SubjectDID, req.SubjectURI, req.ReasonType, req.ReasonText)
	if err != nil {
		logger.Error("Failed to create report: %v", err)
		WriteInternalError(w, "Failed to submit report")
		return
	}

	WriteSuccess(w, map[string]interface{}{"id": id, "status": "ok"})
}

func (m *ModerationHandler) AdminGetReports(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	status := r.URL.Query().Get("status")
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	reports, err := m.db.GetReports(r.Context(), status, limit, offset)
	if err != nil {
		WriteInternalError(w, "Failed to fetch reports")
		return
	}

	uniqueDIDs := make(map[string]bool)
	for _, rpt := range reports {
		uniqueDIDs[rpt.ReporterDID] = true
		uniqueDIDs[rpt.SubjectDID] = true
	}
	dids := make([]string, 0, len(uniqueDIDs))
	for did := range uniqueDIDs {
		dids = append(dids, did)
	}
	profiles := fetchProfilesForDIDs(m.db, dids)

	type HydratedReport struct {
		ID         int     `json:"id"`
		Reporter   Author  `json:"reporter"`
		Subject    Author  `json:"subject"`
		SubjectURI *string `json:"subjectUri,omitempty"`
		ReasonType string  `json:"reasonType"`
		ReasonText *string `json:"reasonText,omitempty"`
		Status     string  `json:"status"`
		CreatedAt  string  `json:"createdAt"`
		ResolvedAt *string `json:"resolvedAt,omitempty"`
		ResolvedBy *string `json:"resolvedBy,omitempty"`
	}

	items := make([]HydratedReport, len(reports))
	for i, rpt := range reports {
		items[i] = HydratedReport{
			ID:         rpt.ID,
			Reporter:   profiles[rpt.ReporterDID],
			Subject:    profiles[rpt.SubjectDID],
			SubjectURI: rpt.SubjectURI,
			ReasonType: rpt.ReasonType,
			ReasonText: rpt.ReasonText,
			Status:     rpt.Status,
			CreatedAt:  rpt.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if rpt.ResolvedAt != nil {
			resolved := rpt.ResolvedAt.Format("2006-01-02T15:04:05Z")
			items[i].ResolvedAt = &resolved
		}
		items[i].ResolvedBy = rpt.ResolvedBy
	}

	pendingCount, _ := m.db.GetReportCount(r.Context(), "pending")
	totalCount, _ := m.db.GetReportCount(r.Context(), "")

	WriteSuccess(w, map[string]interface{}{
		"items":        items,
		"totalItems":   totalCount,
		"pendingCount": pendingCount,
	})
}

func (m *ModerationHandler) AdminTakeAction(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	var req struct {
		ReportID int     `json:"reportId"`
		Action   string  `json:"action"`
		Comment  *string `json:"comment,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	validActions := map[string]bool{
		"acknowledge": true,
		"escalate":    true,
		"takedown":    true,
		"dismiss":     true,
	}

	if !validActions[req.Action] {
		WriteBadRequest(w, "Invalid action")
		return
	}

	report, err := m.db.GetReport(r.Context(), req.ReportID)
	if err != nil {
		WriteNotFound(w, "Report not found")
		return
	}

	if err := m.db.CreateModerationAction(r.Context(), req.ReportID, session.DID, req.Action, req.Comment); err != nil {
		logger.Error("Failed to create moderation action: %v", err)
		WriteInternalError(w, "Failed to take action")
		return
	}

	resolveStatus := "resolved"
	switch req.Action {
	case "dismiss":
		resolveStatus = "dismissed"
	case "escalate":
		resolveStatus = "escalated"
	case "takedown":
		resolveStatus = "resolved"
		if report.SubjectURI != nil && *report.SubjectURI != "" {
			m.deleteContent(r.Context(), *report.SubjectURI)
		}
	case "acknowledge":
		resolveStatus = "acknowledged"
	}

	if err := m.db.ResolveReport(r.Context(), req.ReportID, session.DID, resolveStatus); err != nil {
		logger.Error("Failed to resolve report: %v", err)
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) AdminGetReport(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		WriteBadRequest(w, "Invalid report ID")
		return
	}

	report, err := m.db.GetReport(r.Context(), id)
	if err != nil {
		WriteNotFound(w, "Report not found")
		return
	}

	actions, _ := m.db.GetReportActions(r.Context(), id)

	profiles := fetchProfilesForDIDs(m.db, []string{report.ReporterDID, report.SubjectDID})

	WriteSuccess(w, map[string]interface{}{
		"report":   report,
		"reporter": profiles[report.ReporterDID],
		"subject":  profiles[report.SubjectDID],
		"actions":  actions,
	})
}

func (m *ModerationHandler) AdminCheckAccess(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"isAdmin": false})
		return
	}

	WriteSuccess(w, map[string]bool{"isAdmin": config.Get().IsAdmin(session.DID)})
}

func (m *ModerationHandler) deleteContent(ctx context.Context, uri string) {
	m.db.MarkTakenDown(ctx, uri)
	m.db.Pool().Exec(ctx, "DELETE FROM notes WHERE uri = $1", uri)
	m.db.Pool().Exec(ctx, "DELETE FROM annotations WHERE uri = $1", uri)
	m.db.Pool().Exec(ctx, "DELETE FROM highlights WHERE uri = $1", uri)
	m.db.Pool().Exec(ctx, "DELETE FROM bookmarks WHERE uri = $1", uri)
	m.db.Pool().Exec(ctx, "DELETE FROM replies WHERE uri = $1", uri)
}

func (m *ModerationHandler) AdminCreateLabel(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	var req struct {
		Src string `json:"src"`
		URI string `json:"uri"`
		Val string `json:"val"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Val == "" {
		WriteBadRequest(w, "val is required")
		return
	}

	labelerDID := config.Get().ServiceDID
	if labelerDID == "" {
		WriteInternalError(w, "SERVICE_DID not configured — cannot issue labels")
		return
	}

	targetURI := req.URI
	if targetURI == "" {
		targetURI = req.Src
	}
	if targetURI == "" {
		WriteBadRequest(w, "src or uri is required")
		return
	}

	validLabels := map[string]bool{
		"sexual":     true,
		"nudity":     true,
		"violence":   true,
		"gore":       true,
		"spam":       true,
		"misleading": true,
	}

	if !validLabels[req.Val] {
		WriteBadRequest(w, "Invalid label value. Must be one of: sexual, nudity, violence, gore, spam, misleading")
		return
	}

	if err := m.db.CreateContentLabel(r.Context(), labelerDID, targetURI, req.Val, session.DID); err != nil {
		logger.Error("Failed to create content label: %v", err)
		WriteInternalError(w, "Failed to create label")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) AdminDeleteLabel(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		WriteBadRequest(w, "Invalid label ID")
		return
	}

	if err := m.db.DeleteContentLabel(r.Context(), id); err != nil {
		WriteInternalError(w, "Failed to delete label")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) AdminGetLabels(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	labels, err := m.db.GetAllContentLabels(r.Context(), limit, offset)
	if err != nil {
		WriteInternalError(w, "Failed to fetch labels")
		return
	}

	uniqueDIDs := make(map[string]bool)
	for _, l := range labels {
		uniqueDIDs[l.CreatedBy] = true
		if len(l.Src) > 4 && l.Src[:4] == "did:" {
			uniqueDIDs[l.Src] = true
		}
	}
	dids := make([]string, 0, len(uniqueDIDs))
	for did := range uniqueDIDs {
		dids = append(dids, did)
	}
	profiles := fetchProfilesForDIDs(m.db, dids)

	type HydratedLabel struct {
		ID        int     `json:"id"`
		Src       string  `json:"src"`
		URI       string  `json:"uri"`
		Val       string  `json:"val"`
		CreatedBy Author  `json:"createdBy"`
		CreatedAt string  `json:"createdAt"`
		Subject   *Author `json:"subject,omitempty"`
	}

	items := make([]HydratedLabel, len(labels))
	for i, l := range labels {
		items[i] = HydratedLabel{
			ID:        l.ID,
			Src:       l.Src,
			URI:       l.URI,
			Val:       l.Val,
			CreatedBy: profiles[l.CreatedBy],
			CreatedAt: l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if len(l.Src) > 4 && l.Src[:4] == "did:" {
			subj := profiles[l.Src]
			items[i].Subject = &subj
		}
	}

	WriteSuccess(w, map[string]interface{}{"items": items})
}

func (m *ModerationHandler) getViewerDID(r *http.Request) string {
	cookie, err := r.Cookie("margin_session")
	if err != nil {
		return ""
	}
	sess, err := m.db.GetOAuthSessionByID(r.Context(), cookie.Value)
	if err != nil {
		return ""
	}
	return sess.DID
}

func (m *ModerationHandler) GetLabelerInfo(w http.ResponseWriter, r *http.Request) {
	serviceDID := config.Get().ServiceDID

	type LabelDefinition struct {
		Identifier  string `json:"identifier"`
		Severity    string `json:"severity"`
		Blurs       string `json:"blurs"`
		Description string `json:"description"`
	}

	labels := []LabelDefinition{
		{Identifier: "sexual", Severity: "inform", Blurs: "content", Description: "Sexual content"},
		{Identifier: "nudity", Severity: "inform", Blurs: "content", Description: "Nudity"},
		{Identifier: "violence", Severity: "inform", Blurs: "content", Description: "Violence"},
		{Identifier: "gore", Severity: "alert", Blurs: "content", Description: "Graphic/gory content"},
		{Identifier: "spam", Severity: "inform", Blurs: "content", Description: "Spam or unwanted content"},
		{Identifier: "misleading", Severity: "inform", Blurs: "content", Description: "Misleading information"},
	}

	WriteSuccess(w, map[string]interface{}{
		"did":    serviceDID,
		"name":   "Margin Moderation",
		"labels": labels,
	})
}

func (m *ModerationHandler) AdminBanAccount(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}
	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	var req struct {
		DID    string  `json:"did"`
		Reason *string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DID == "" {
		WriteBadRequest(w, "did is required")
		return
	}

	if err := m.db.BanAccount(r.Context(), req.DID, session.DID, req.Reason); err != nil {
		logger.Error("Failed to ban account: %v", err)
		WriteInternalError(w, "Failed to ban account")
		return
	}

	m.db.DeleteOAuthSessionsByDID(r.Context(), req.DID)

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) AdminUnbanAccount(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}
	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	did := r.URL.Query().Get("did")
	if did == "" {
		WriteBadRequest(w, "did is required")
		return
	}

	if err := m.db.UnbanAccount(r.Context(), did); err != nil {
		logger.Error("Failed to unban account: %v", err)
		WriteInternalError(w, "Failed to unban account")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}

func (m *ModerationHandler) AdminGetBannedAccounts(w http.ResponseWriter, r *http.Request) {
	session, err := m.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}
	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "Forbidden")
		return
	}

	accounts, err := m.db.GetBannedAccounts(r.Context())
	if err != nil {
		WriteInternalError(w, "Failed to get banned accounts")
		return
	}

	if accounts == nil {
		accounts = []db.BannedAccount{}
	}

	dids := make([]string, len(accounts))
	for i, a := range accounts {
		dids[i] = a.DID
	}
	profileMap := fetchProfilesForDIDs(m.db, dids)

	type BannedEntry struct {
		db.BannedAccount
		Profile *Author `json:"profile,omitempty"`
	}
	entries := make([]BannedEntry, len(accounts))
	for i, a := range accounts {
		p, ok := profileMap[a.DID]
		var profile *Author
		if ok {
			profile = &p
		}
		entries[i] = BannedEntry{BannedAccount: a, Profile: profile}
	}

	WriteSuccess(w, map[string]interface{}{"items": entries, "total": len(entries)})
}
