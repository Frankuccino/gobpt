package email

func (c *Client) SendWelcomeEmail(to, firstName string) error {
	data := map[string]string{
		"UserFirstName": firstName,
	}

	return c.SendEmail(
		to,
		"Welcome to boilerplate!",
		TemplateWelcome,
		data,
	)
}

// This is a utility function that'll make it easier to call and keep scope limited to this package.
