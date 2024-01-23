// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package mail

import (
	"bytes"
	"io"
	"net/http"
	"slices"
	"time"

	form_service "github.com/denyskon/postalion/form"
	mail "github.com/xhit/go-simple-mail/v2"
)

type MailAccount struct {
	Host       string `validate:"required,fqdn|hostname|ip"`
	Port       int    `validate:"required"`
	Username   string `validate:"required"`
	Password   string `validate:"required"`
	Encryption string `validate:"oneof=ssl starttls none"`
}

func (a MailAccount) Init() (*mail.SMTPServer, error) {
	server := mail.NewSMTPClient()
	server.Host = a.Host
	server.Port = a.Port
	server.Username = a.Username
	server.Password = a.Password
	switch a.Encryption {
	case "ssl":
		{
			server.Encryption = mail.EncryptionSSLTLS
		}
	case "starttls":
		{
			server.Encryption = mail.EncryptionSTARTTLS
		}
	default:
		{
			server.Encryption = mail.EncryptionNone
		}
	}
	server.Authentication = mail.AuthAuto
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second
	smtpClient, err := server.Connect()
	if err != nil {
		return nil, err
	}

	smtpClient.Close()
	return server, nil
}

func ParseForm(form *form_service.Form, req *http.Request) ([]*form_service.StringField, []*mail.File, error) {
	fieldsToParse := slices.DeleteFunc[[]form_service.Field, form_service.Field](form.MailConfig.Fields, func(field form_service.Field) bool {
		return form.MailConfig.SubjectField == field.Name
	})

	var formContent []*form_service.StringField
	var formFiles []*mail.File

	for _, field := range fieldsToParse {
		switch field.Type {
		case "string":
			{

				formContent = append(formContent, &form_service.StringField{
					Name:    field.Name,
					Label:   field.Label,
					Content: req.FormValue(field.Name),
				})
			}
		case "file":
			{
				file, fileHeader, err := req.FormFile(field.Name)
				if err != nil {
					if err == http.ErrMissingFile {
						continue
					}
					return nil, nil, err
				}
				defer file.Close()

				buf := bytes.NewBuffer(nil)
				if _, err := io.Copy(buf, file); err != nil {
					return nil, nil, err
				}
				formFiles = append(formFiles, &mail.File{
					Name:     fileHeader.Filename,
					MimeType: fileHeader.Header["Content-Type"][0],
					Data:     buf.Bytes(),
				})
			}
		}
	}

	return formContent, formFiles, nil
}

func GetSubject(form *form_service.Form, req *http.Request) string {
	if form.MailConfig.SubjectField != "" {
		return req.FormValue(form.MailConfig.SubjectField)
	}

	return form.MailConfig.Subject
}

func HandleForm(form *form_service.Form, req *http.Request, account *mail.SMTPServer) error {
	formContent, formFiles, err := ParseForm(form, req)
	if err != nil {
		return err
	}

	client, err := account.Connect()
	if err != nil {
		return err
	}

	body, err := form.HTML(formContent)
	if err != nil {
		return err
	}

	message := mail.NewMSG().
		SetFrom(form.MailConfig.From).
		AddTo(form.MailConfig.To).
		SetSubject(GetSubject(form, req)).
		SetBody(mail.TextHTML, body)

	if replyTo := form.MailConfig.ReplyTo.GetFormattedString(req); replyTo != "" {
		message.SetReplyTo(replyTo)
	}

	for _, file := range formFiles {
		message.Attach(file)
	}

	if message.Error != nil {
		return err
	}

	if err := message.Send(client); err != nil {
		return err
	}

	return nil
}
