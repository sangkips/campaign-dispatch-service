package campaigns

import (
	"database/sql"
	"testing"

	customersModels "github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
)

// TestRenderTemplate_AllFieldsPresent tests template rendering when all customer fields are populated
func TestRenderTemplate_AllFieldsPresent(t *testing.T) {
	template := "Hello {first_name} {last_name}! Your phone is {phone}. You're in {location} and prefer {prefered_product}."

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        1,
		Firstname: "John",
		Lastname:  "Doe",
		Phone:     "+254712345678",
		Location: sql.NullString{
			String: "Nairobi",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Premium Plan",
			Valid:  true,
		},
	}

	expected := "Hello John Doe! Your phone is +254712345678. You're in Nairobi and prefer Premium Plan."
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_NullLocation tests template rendering when location is null
// Behavior: Null location fields are replaced with empty string
func TestRenderTemplate_NullLocation(t *testing.T) {
	template := "Hello {first_name}! Location: {location}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        2,
		Firstname: "Jane",
		Lastname:  "Smith",
		Phone:     "+254723456789",
		Location: sql.NullString{
			Valid: false, // NULL location
		},
		PreferedProduct: sql.NullString{
			String: "Basic Plan",
			Valid:  true,
		},
	}

	expected := "Hello Jane! Location: "
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_NullPreferedProduct tests template rendering when preferred product is null
// Behavior: Null preferred product fields are replaced with empty string
func TestRenderTemplate_NullPreferedProduct(t *testing.T) {
	template := "Hi {first_name}! Your preferred product: {prefered_product}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        3,
		Firstname: "Alice",
		Lastname:  "Johnson",
		Phone:     "+254734567890",
		Location: sql.NullString{
			String: "Mombasa",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			Valid: false, // NULL preferred product
		},
	}

	expected := "Hi Alice! Your preferred product: "
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_AllNullableFieldsNull tests template rendering when all nullable fields are null
// Behavior: All null fields are replaced with empty strings
func TestRenderTemplate_AllNullableFieldsNull(t *testing.T) {
	template := "Hello {first_name} {last_name}! Phone: {phone}, Location: {location}, Product: {prefered_product}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        4,
		Firstname: "Bob",
		Lastname:  "Williams",
		Phone:     "+254745678901",
		Location: sql.NullString{
			Valid: false, // NULL
		},
		PreferedProduct: sql.NullString{
			Valid: false, // NULL
		},
	}

	expected := "Hello Bob Williams! Phone: +254745678901, Location: , Product: "
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_MultipleOccurrences tests that all occurrences of placeholders are replaced
func TestRenderTemplate_MultipleOccurrences(t *testing.T) {
	template := "{first_name}, {first_name}, {first_name}! Your name is {first_name}."

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        5,
		Firstname: "Charlie",
		Lastname:  "Brown",
		Phone:     "+254756789012",
		Location: sql.NullString{
			String: "Kisumu",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Standard",
			Valid:  true,
		},
	}

	expected := "Charlie, Charlie, Charlie! Your name is Charlie."
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_NoPlaceholders tests template without any placeholders
func TestRenderTemplate_NoPlaceholders(t *testing.T) {
	template := "This is a static message with no placeholders."

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        6,
		Firstname: "David",
		Lastname:  "Miller",
		Phone:     "+254767890123",
		Location: sql.NullString{
			String: "Nakuru",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Pro",
			Valid:  true,
		},
	}

	expected := "This is a static message with no placeholders."
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_EmptyTemplate tests rendering with an empty template
func TestRenderTemplate_EmptyTemplate(t *testing.T) {
	template := ""

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        7,
		Firstname: "Emma",
		Lastname:  "Davis",
		Phone:     "+254778901234",
		Location: sql.NullString{
			String: "Eldoret",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Enterprise",
			Valid:  true,
		},
	}

	expected := ""
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_Preferred_product {preferred_product} placeholder
func TestRenderTemplate_Preferred_product(t *testing.T) {
	template := "Your preferred product is: {prefered_product}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        8,
		Firstname: "Frank",
		Lastname:  "Wilson",
		Phone:     "+254789012345",
		Location: sql.NullString{
			String: "Thika",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Gold Package",
			Valid:  true,
		},
	}

	expected := "Your preferred product is: Gold Package"
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_SpecialCharactersInData tests customer data with special characters
func TestRenderTemplate_SpecialCharactersInData(t *testing.T) {
	template := "Hello {first_name}! Location: {location}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        10,
		Firstname: "O'Brien",
		Lastname:  "Smith-Jones",
		Phone:     "+254701234567",
		Location: sql.NullString{
			String: "Nairobi (CBD)",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Plan A & B",
			Valid:  true,
		},
	}

	expected := "Hello O'Brien! Location: Nairobi (CBD)"
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_EmptyStringFields tests customer with empty string values (not null)
func TestRenderTemplate_EmptyStringFields(t *testing.T) {
	template := "Name: {first_name} {last_name}, Phone: {phone}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        11,
		Firstname: "",
		Lastname:  "",
		Phone:     "+254712345678",
		Location: sql.NullString{
			String: "",
			Valid:  true, // Empty but valid
		},
		PreferedProduct: sql.NullString{
			String: "",
			Valid:  true, // Empty but valid
		},
	}

	expected := "Name:  , Phone: +254712345678"
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestRenderTemplate_MixedCaseInTemplate tests that placeholders are case-sensitive
func TestRenderTemplate_CaseSensitive(t *testing.T) {
	template := "Hello {first_name} and {FIRST_NAME} and {First_Name}"

	customer := customersModels.GetCustomerForPreviewRow{
		ID:        12,
		Firstname: "Henry",
		Lastname:  "Anderson",
		Phone:     "+254723456789",
		Location: sql.NullString{
			String: "Nairobi",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Basic",
			Valid:  true,
		},
	}

	// Only {first_name} should be replaced, others should remain unchanged
	expected := "Hello Henry and {FIRST_NAME} and {First_Name}"
	result := RenderTemplate(template, customer)

	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

// TestToCustomerPreviewData_AllFieldsPresent tests conversion with all fields present
func TestToCustomerPreviewData_AllFieldsPresent(t *testing.T) {
	customer := customersModels.GetCustomerForPreviewRow{
		ID:        1,
		Firstname: "John",
		Lastname:  "Doe",
		Phone:     "+254712345678",
		Location: sql.NullString{
			String: "Nairobi",
			Valid:  true,
		},
		PreferedProduct: sql.NullString{
			String: "Premium",
			Valid:  true,
		},
	}

	result := ToCustomerPreviewData(customer)

	if result.ID != 1 {
		t.Errorf("ID = %d, want %d", result.ID, 1)
	}
	if result.FirstName != "John" {
		t.Errorf("FirstName = %q, want %q", result.FirstName, "John")
	}
	if result.LastName != "Doe" {
		t.Errorf("LastName = %q, want %q", result.LastName, "Doe")
	}
	if result.Phone != "+254712345678" {
		t.Errorf("Phone = %q, want %q", result.Phone, "+254712345678")
	}
	if result.Location == nil {
		t.Error("Location should not be nil")
	} else if *result.Location != "Nairobi" {
		t.Errorf("Location = %q, want %q", *result.Location, "Nairobi")
	}
	if result.PreferedProduct == nil {
		t.Error("PreferedProduct should not be nil")
	} else if *result.PreferedProduct != "Premium" {
		t.Errorf("PreferedProduct = %q, want %q", *result.PreferedProduct, "Premium")
	}
}

// TestToCustomerPreviewData_NullFields tests conversion with null fields
func TestToCustomerPreviewData_NullFields(t *testing.T) {
	customer := customersModels.GetCustomerForPreviewRow{
		ID:        2,
		Firstname: "Jane",
		Lastname:  "Smith",
		Phone:     "+254723456789",
		Location: sql.NullString{
			Valid: false,
		},
		PreferedProduct: sql.NullString{
			Valid: false,
		},
	}

	result := ToCustomerPreviewData(customer)

	if result.ID != 2 {
		t.Errorf("ID = %d, want %d", result.ID, 2)
	}
	if result.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want %q", result.FirstName, "Jane")
	}
	if result.LastName != "Smith" {
		t.Errorf("LastName = %q, want %q", result.LastName, "Smith")
	}
	if result.Phone != "+254723456789" {
		t.Errorf("Phone = %q, want %q", result.Phone, "+254723456789")
	}
	if result.Location != nil {
		t.Errorf("Location should be nil, got %q", *result.Location)
	}
	if result.PreferedProduct != nil {
		t.Errorf("PreferedProduct should be nil, got %q", *result.PreferedProduct)
	}
}

// TestToCustomerPreviewData_PartialNullFields tests conversion with only location null
func TestToCustomerPreviewData_PartialNullFields(t *testing.T) {
	customer := customersModels.GetCustomerForPreviewRow{
		ID:        3,
		Firstname: "Alice",
		Lastname:  "Johnson",
		Phone:     "+254734567890",
		Location: sql.NullString{
			Valid: false,
		},
		PreferedProduct: sql.NullString{
			String: "Standard",
			Valid:  true,
		},
	}

	result := ToCustomerPreviewData(customer)

	if result.Location != nil {
		t.Errorf("Location should be nil, got %q", *result.Location)
	}
	if result.PreferedProduct == nil {
		t.Error("PreferedProduct should not be nil")
	} else if *result.PreferedProduct != "Standard" {
		t.Errorf("PreferedProduct = %q, want %q", *result.PreferedProduct, "Standard")
	}
}
