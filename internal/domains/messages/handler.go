package messages

import (
	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
)

type Handler struct {
	svc *Service
}

func NewHandler(db models.DBTX) *Handler {
	repo := NewRepository(db)
	return &Handler{svc: NewService(repo)}
}

