package campaigns

import (
	"strings"

	customersModels "github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
)

// RenderTemplate replaces template variables with customer data
// Supported variables: {first_name}, {last_name}, {location}, {prefered_product}, {phone}
func RenderTemplate(template string, customer customersModels.GetCustomerForPreviewRow) string {
	rendered := template

	// Replace {first_name}
	rendered = strings.ReplaceAll(rendered, "{first_name}", customer.Firstname)

	// Replace {last_name}
	rendered = strings.ReplaceAll(rendered, "{last_name}", customer.Lastname)

	// Replace {phone}
	rendered = strings.ReplaceAll(rendered, "{phone}", customer.Phone)

	// Replace {location} - handle nullable field
	if customer.Location.Valid {
		rendered = strings.ReplaceAll(rendered, "{location}", customer.Location.String)
	} else {
		rendered = strings.ReplaceAll(rendered, "{location}", "")
	}

	// Replace {prefered_product} - handle nullable field
	if customer.PreferedProduct.Valid {
		rendered = strings.ReplaceAll(rendered, "{prefered_product}", customer.PreferedProduct.String)
	} else {
		rendered = strings.ReplaceAll(rendered, "{prefered_product}", "")
	}

	return rendered
}

// CustomerPreviewData converts GetCustomerForPreviewRow to a JSON-friendly format
type CustomerPreviewData struct {
	ID              int32   `json:"id"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Phone           string  `json:"phone"`
	Location        *string `json:"location,omitempty"`
	PreferedProduct *string `json:"prefered_product,omitempty"`
}

// ToCustomerPreviewData converts the database model to JSON-friendly format
func ToCustomerPreviewData(customer customersModels.GetCustomerForPreviewRow) CustomerPreviewData {
	data := CustomerPreviewData{
		ID:        customer.ID,
		FirstName: customer.Firstname,
		LastName:  customer.Lastname,
		Phone:     customer.Phone,
	}

	if customer.Location.Valid {
		location := customer.Location.String
		data.Location = &location
	}

	if customer.PreferedProduct.Valid {
		product := customer.PreferedProduct.String
		data.PreferedProduct = &product
	}

	return data
}

