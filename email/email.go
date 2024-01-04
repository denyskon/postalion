// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package email

import (
	"time"

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
