package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"margin.at/internal/config"
	"margin.at/internal/db"
	"margin.at/internal/xrpc"
)

type LabelerSubscription struct {
	DID string `json:"did"`
}

type LabelPreference struct {
	LabelerDID string `json:"labelerDid"`
	Label      string `json:"label"`
	Visibility string `json:"visibility"`
}

type PreferencesResponse struct {
	ExternalLinkSkippedHostnames []string              `json:"externalLinkSkippedHostnames"`
	SubscribedLabelers           []LabelerSubscription `json:"subscribedLabelers"`
	LabelPreferences             []LabelPreference     `json:"labelPreferences"`
	DisableExternalLinkWarning   bool                  `json:"disableExternalLinkWarning"`
	EnableCommunityBookmarks     bool                  `json:"enableCommunityBookmarks"`
}

func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	prefs, err := h.db.GetPreferences(r.Context(), session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to fetch preferences")
		return
	}

	hostnames := []string{}
	if prefs != nil && prefs.ExternalLinkSkippedHostnames != nil {
		json.Unmarshal([]byte(*prefs.ExternalLinkSkippedHostnames), &hostnames)
	}

	var labelers []LabelerSubscription
	if prefs != nil && prefs.SubscribedLabelers != nil {
		json.Unmarshal([]byte(*prefs.SubscribedLabelers), &labelers)
	}
	if labelers == nil {
		labelers = []LabelerSubscription{}
		serviceDID := config.Get().ServiceDID
		if serviceDID != "" {
			labelers = append(labelers, LabelerSubscription{DID: serviceDID})
		}
	}

	var labelPrefs []LabelPreference
	if prefs != nil && prefs.LabelPreferences != nil {
		json.Unmarshal([]byte(*prefs.LabelPreferences), &labelPrefs)
	}
	if labelPrefs == nil {
		labelPrefs = []LabelPreference{}
	}

	disableWarning := false
	if prefs != nil && prefs.DisableExternalLinkWarning != nil {
		disableWarning = *prefs.DisableExternalLinkWarning
	}

	enableCommunityBookmarks := true
	if prefs != nil && prefs.EnableCommunityBookmarks != nil {
		enableCommunityBookmarks = *prefs.EnableCommunityBookmarks
	}

	WriteSuccess(w, PreferencesResponse{
		ExternalLinkSkippedHostnames: hostnames,
		SubscribedLabelers:           labelers,
		LabelPreferences:             labelPrefs,
		DisableExternalLinkWarning:   disableWarning,
		EnableCommunityBookmarks:     enableCommunityBookmarks,
	})
}

func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	var input PreferencesResponse
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, "Invalid input")
		return
	}

	var xrpcLabelers []xrpc.LabelerSubscription
	for _, l := range input.SubscribedLabelers {
		xrpcLabelers = append(xrpcLabelers, xrpc.LabelerSubscription{
			Type: "at.margin.preferences#labelerSubscription",
			DID:  l.DID,
		})
	}
	var xrpcLabelPrefs []xrpc.LabelPreference
	for _, lp := range input.LabelPreferences {
		xrpcLabelPrefs = append(xrpcLabelPrefs, xrpc.LabelPreference{
			Type:       "at.margin.preferences#labelPreference",
			LabelerDID: lp.LabelerDID,
			Label:      lp.Label,
			Visibility: lp.Visibility,
		})
	}

	record := xrpc.NewPreferencesRecord(input.ExternalLinkSkippedHostnames, xrpcLabelers, xrpcLabelPrefs, &input.DisableExternalLinkWarning, &input.EnableCommunityBookmarks)
	if err := record.Validate(); err != nil {
		WriteBadRequest(w, fmt.Sprintf("Invalid record: %v", err))
		return
	}

	err = h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := client.PutRecord(ctx, did, xrpc.CollectionPreferences, "self", record)
		return err
	})

	if err != nil {
		fmt.Printf("[UpdatePreferences] PDS write failed: %v\n", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)
	hostnamesJSON, _ := json.Marshal(input.ExternalLinkSkippedHostnames)
	hostnamesStr := string(hostnamesJSON)

	var subscribedLabelersPtr, labelPrefsPtr *string
	if len(input.SubscribedLabelers) > 0 {
		labelersJSON, _ := json.Marshal(input.SubscribedLabelers)
		s := string(labelersJSON)
		subscribedLabelersPtr = &s
	}
	if len(input.LabelPreferences) > 0 {
		prefsJSON, _ := json.Marshal(input.LabelPreferences)
		s := string(prefsJSON)
		labelPrefsPtr = &s
	}

	uri := fmt.Sprintf("at://%s/%s/self", session.DID, xrpc.CollectionPreferences)

	err = h.db.UpsertPreferences(r.Context(), &db.Preferences{
		URI:                          uri,
		AuthorDID:                    session.DID,
		ExternalLinkSkippedHostnames: &hostnamesStr,
		SubscribedLabelers:           subscribedLabelersPtr,
		LabelPreferences:             labelPrefsPtr,
		DisableExternalLinkWarning:   &input.DisableExternalLinkWarning,
		EnableCommunityBookmarks:     &input.EnableCommunityBookmarks,
		CreatedAt:                    createdAt,
		IndexedAt:                    time.Now(),
	})

	if err != nil {
		fmt.Printf("Failed to update local db preferences: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}
