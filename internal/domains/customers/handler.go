package customers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
	"github.com/sangkips/campaign-dispatch-service/internal/handlers"
)

type Handler struct {
	svc *Service
}

func NewHandler(db models.DBTX) *Handler {
	repo := NewRepository(db)
	return &Handler{svc: NewService(repo)}
}

func (h *Handler) RegisterCustomerRoutes(r chi.Router) {
	r.Post("/", h.createCustomer)
}

func (h *Handler) createCustomer(w http.ResponseWriter, r *http.Request) {
	var req CreateCustomerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	ctx := r.Context()

	customer, err := h.svc.repo.CreateCustomer(ctx, models.CreateCustomerParams{
		Phone:           req.Phone,
		Firstname:       req.Firstname,
		Lastname:        req.Lastname,
		Location:        stringToNullString(req.Location),
		PreferedProduct: stringToNullString(req.PreferedProduct),
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create customer")
		handlers.RespondWithError(w, http.StatusInternalServerError, "CUSTOMER_CREATE_FAILED", "Failed to create customer: "+err.Error())
		return
	}

	handlers.RespondWithJSON(w, http.StatusCreated, customer)

}

func stringToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
