package campaigns

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns/models"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/customers"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages"
	"github.com/sangkips/campaign-dispatch-service/internal/handlers"
)

type Handler struct {
	svc *Service
}

func NewHandler(db models.DBTX, queue QueuePublisher) *Handler {
	campaignRepo := NewRepository(db)
	messagesRepo := messages.NewRepository(db)
	customersRepo := customers.NewRepository(db)
	return &Handler{svc: NewService(campaignRepo, messagesRepo, customersRepo, queue)}
}

func (h *Handler) RegisterCampaignRoutes(r chi.Router) {
	r.Post("/", h.createCampaign)
	r.Post("/{id}/send", h.sendCampaign)
	r.Post("/{id}/personalized-preview", h.personalizedPreview)
	r.Get("/", h.listCampaigns)
	r.Get("/{id}", h.getCampaign)
}

// Helper function to convert *time.Time to sql.NullTime
func timeToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func (h *Handler) createCampaign(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	ctx := r.Context()

	params := models.CreateCampaignParams{
		Name:         req.Name,
		Channel:      req.Channel,
		ScheduledAt:  timeToNullTime(req.ScheduledAt),
		BaseTemplate: req.BaseTemplate,
	}

	campaign, err := h.svc.repo.CreateCampaign(ctx, params)
	if err != nil {
		handlers.RespondWithError(w, http.StatusInternalServerError, "CAMPAIGN_CREATE_FAILED", "Failed to create campaign: "+err.Error())
		return
	}

	handlers.RespondWithJSON(w, http.StatusCreated, campaign)

}

func (h *Handler) sendCampaign(w http.ResponseWriter, r *http.Request) {
	// Get campaign ID from URL
	campaignIDStr := chi.URLParam(r, "id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 32)
	if err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_CAMPAIGN_ID", "Invalid campaign ID format")
		return
	}

	// Parse request body
	var req SendCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Call service to send campaign
	response, err := h.svc.SendCampaign(ctx, int32(campaignID), req)
	if err != nil {
		// Determine appropriate status code and error code based on error
		if err.Error() == "campaign not found" {
			handlers.RespondWithError(w, http.StatusNotFound, "CAMPAIGN_NOT_FOUND", "Campaign with ID "+campaignIDStr+" not found")
		} else if err.Error() == "customer_ids cannot be empty" {
			handlers.RespondWithError(w, http.StatusBadRequest, "EMPTY_CUSTOMER_IDS", "customer_ids cannot be empty")
		} else if err.Error() == "campaign must be in draft or scheduled status" {
			handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_CAMPAIGN_STATUS", "Campaign must be in draft or scheduled status")
		} else {
			handlers.RespondWithError(w, http.StatusInternalServerError, "CAMPAIGN_SEND_FAILED", "Failed to send campaign: "+err.Error())
		}
		return
	}

	handlers.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) listCampaigns(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	channel := r.URL.Query().Get("channel")
	status := r.URL.Query().Get("status")

	page := int32(1)
	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
			page = int32(p)
		}
	}

	pageSize := int32(20)
	if pageSizeStr != "" {
		if ps, err := strconv.ParseInt(pageSizeStr, 10, 32); err == nil {
			pageSize = int32(ps)
		}
	}

	params := ListCampaignsParams{
		Page:     page,
		PageSize: pageSize,
		Channel:  channel,
		Status:   status,
	}

	response, err := h.svc.ListCampaigns(r.Context(), params)
	if err != nil {
		handlers.RespondWithError(w, http.StatusInternalServerError, "CAMPAIGNS_LIST_FAILED", "Failed to list campaigns: "+err.Error())
		return
	}

	handlers.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) getCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_CAMPAIGN_ID", "Invalid campaign ID format")
		return
	}

	response, err := h.svc.GetCampaign(r.Context(), int32(id))
	if err != nil {
		if err == sql.ErrNoRows {
			handlers.RespondWithError(w, http.StatusNotFound, "CAMPAIGN_NOT_FOUND", "Campaign with ID "+idStr+" not found")
			return
		}
		handlers.RespondWithError(w, http.StatusInternalServerError, "CAMPAIGN_GET_FAILED", "Failed to get campaign: "+err.Error())
		return
	}

	handlers.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) personalizedPreview(w http.ResponseWriter, r *http.Request) {
	// Get campaign ID from URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_CAMPAIGN_ID", "Invalid campaign ID format")
		return
	}

	// Parse request body
	var req PersonalizedPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	// Validate customer_id is provided
	if req.CustomerID == 0 {
		handlers.RespondWithError(w, http.StatusBadRequest, "MISSING_CUSTOMER_ID", "customer_id is required")
		return
	}

	// Call service to generate preview
	response, err := h.svc.PersonalizedPreview(r.Context(), int32(id), req)
	if err != nil {
		// Determine appropriate error code based on error
		if err.Error() == "campaign not found" {
			handlers.RespondWithError(w, http.StatusNotFound, "CAMPAIGN_NOT_FOUND", "Campaign with ID "+idStr+" not found")
		} else if err.Error() == "customer not found" {
			handlers.RespondWithError(w, http.StatusNotFound, "CUSTOMER_NOT_FOUND", "Customer not found")
		} else {
			handlers.RespondWithError(w, http.StatusInternalServerError, "PREVIEW_FAILED", "Failed to generate preview: "+err.Error())
		}
		return
	}

	handlers.RespondWithJSON(w, http.StatusOK, response)
}
