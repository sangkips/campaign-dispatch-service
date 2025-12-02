package customers

import (
	"context"

	"github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
)

type Repository interface {
	CreateCustomer(ctx context.Context, customer models.CreateCustomerParams) (models.Customer, error)
	GetCustomerForPreview(ctx context.Context, id int32) (models.GetCustomerForPreviewRow, error)
}

type repository struct {
	q *models.Queries
}

func NewRepository(db models.DBTX) Repository {
	return &repository{q: models.New(db)}
}

func (r *repository) CreateCustomer(ctx context.Context, customer models.CreateCustomerParams) (models.Customer, error) {
	return r.q.CreateCustomer(ctx, customer)
}

func (r *repository) GetCustomerForPreview(ctx context.Context, id int32) (models.GetCustomerForPreviewRow, error) {
	return r.q.GetCustomerForPreview(ctx, id)
}
