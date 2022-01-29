package unifi

import "fmt"

type Controller struct {
	username string
	site     string
}

func NewController(username, site string) *Controller {
	return &Controller{
		username: username,
		site:     site,
	}
}

func (c *Controller) Login(password string) error {
	return fmt.Errorf("unimplemented")
}
