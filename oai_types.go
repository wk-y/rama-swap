package main

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"` // must equal "model"
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}
