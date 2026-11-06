package requests

type LoginAppleInput struct {
	IDToken string `json:"id_token"`
	Name    string `json:"name"`
}
