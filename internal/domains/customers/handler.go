package customers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	r.Get("/", h.listCustomers)
}

// CustomerResponse is the API response format for customers
type CustomerResponse struct {
	ID              int32   `json:"id"`
	Phone           string  `json:"phone"`
	Firstname       string  `json:"firstname"`
	Lastname        string  `json:"lastname"`
	Location        *string `json:"location,omitempty"`
	PreferedProduct *string `json:"prefered_product,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

// toCustomerResponse converts a models.Customer to CustomerResponse
func toCustomerResponse(customer models.Customer) CustomerResponse {
	resp := CustomerResponse{
		ID:        customer.ID,
		Phone:     customer.Phone,
		Firstname: customer.Firstname,
		Lastname:  customer.Lastname,
		CreatedAt: customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if customer.Location.Valid {
		resp.Location = &customer.Location.String
	}

	if customer.PreferedProduct.Valid {
		resp.PreferedProduct = &customer.PreferedProduct.String
	}

	return resp
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

func (h *Handler) listCustomers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int32(100)
	offset := int32(0)

	if limitStr != "" {
		if parsedLimit, err := parseInt32(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := parseInt32(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	customers, err := h.svc.repo.ListCustomers(ctx, models.ListCustomersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list customers")
		handlers.RespondWithError(w, http.StatusInternalServerError, "CUSTOMERS_LIST_FAILED", "Failed to list customers: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		response[i] = toCustomerResponse(customer)
	}

	handlers.RespondWithJSON(w, http.StatusOK, response)
}

func parseInt32(s string) (int32, error) {
	var result int32
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func stringToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
