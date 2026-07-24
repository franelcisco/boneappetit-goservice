package mailgun

import (
	"context"
	"time"

	"github.com/mailgun/mailgun-go/v5"
	"go.uber.org/zap"
)

type Repository interface {
	SendEmail(ctx context.Context, req SendEmailRequest) error
}

type repository struct {
	client *MailGunClient
	domain string
	sender string
	logger *zap.Logger
}

func NewRepository(client *MailGunClient, domain string, sender string, logger *zap.Logger) Repository {
	return &repository{
		client: client,
		domain: domain,
		sender: sender,
		logger: logger,
	}
}

func (r *repository) SendEmail(ctx context.Context, req SendEmailRequest) error {
	// The message object allows you to add attachments and Bcc recipients
	message := mailgun.NewMessage(r.domain, r.sender, req.Subject, req.Body, req.To)

	if req.Template != "" {
		message.SetTemplate(req.Template)
	}

	// Set email variables if any
	if len(req.Vars) > 0 {
		err := r.setEmailVariables(message, req.Vars)
		if err != nil {
			r.logger.Error(err.Error(), zap.String("to", req.To), zap.String("template", req.Template))
			return err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	// Send the message with a 10-second timeout
	_, err := r.client.mg.Send(ctx, message)

	if err != nil {
		r.logger.Error(err.Error(), zap.String("to", req.To), zap.String("template", req.Template))
		return err
	}

	//r.logger.Info("Email sent successfully", zap.String("to", req.To), zap.String("template", req.Template))
	return nil
}

// setEmailVariables sets the variables for the email template
func (r *repository) setEmailVariables(message *mailgun.PlainMessage, vars map[string]any) error {

	for key, value := range vars {
		message.AddVariable(key, value)
	}

	return nil
}
