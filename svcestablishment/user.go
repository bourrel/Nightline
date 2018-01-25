package svcestablishment

type User struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	Pseudo        string `json:"pseudo"`
	Password      string `json:"password"`
	Token         string `json:"token"`
	Firstname     string `json:"firstname"`
	Surname       string `json:"surname"`
	Number        string `json:"number"`
	Image         string `json:"image"`
	SuccessPoints int64  `json:"success_points"`
	ConnectedTo   []User `json:"connected_to"`
}
