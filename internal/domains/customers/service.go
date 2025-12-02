package customers

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type CreateCustomerRequest struct {
	Phone           string  `json:"phone"`
	Firstname       string  `json:"firstname"`
	Lastname        string  `json:"lastname"`
	Location        *string `json:"location"`
	PreferedProduct *string `json:"prefered_product"`
}
