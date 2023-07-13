package authDto

type SignUpRequest struct {
	MobileNumber string `json:"mobile_number" validate:"required,min=9"`
	FirstName    string `json:"first_name" validate:"required,min=3"`
	LastName     string `json:"last_name" validate:"required,min=3"`
	Password     string `json:"password" validate:"required,min=8"`
	Code         string `json:"code"`
}
type VerifyRequest struct {
	MobileNumber string `json:"mobileNumber" validation:"required"`
	Code         string `json:"code" validation:"required"`
}
type LoginRequest struct {
	MobileNumber string `json:"mobileNumber"`
	Password     string `json:"password"`
}
type ForgotRequest struct {
	Email string `json:"email"`
}