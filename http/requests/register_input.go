package requests

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=3,max=255"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=6,max=255"`
}
