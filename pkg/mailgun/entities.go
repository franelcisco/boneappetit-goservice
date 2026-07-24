package mailgun

type SendEmailRequest struct {
	To       string
	Subject  string
	Body     string
	Template string
	Vars     map[string]any
}
