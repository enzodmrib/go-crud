package models

type User struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Biography *string `json:"biography"`
}
