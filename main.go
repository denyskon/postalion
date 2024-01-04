// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package main

import (
	"fmt"
	"net/http"

	mail_services "github.com/denyskon/postalion/email"
	form_service "github.com/denyskon/postalion/form"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/pat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Config struct {
	MailAccounts map[string]*mail_services.MailAccount `validate:"required,dive,required"`
	Forms        map[string]*form_service.Form         `validate:"required,dive,required"`
	Port         int                                   `validate:"required"`
	TemplateDir  string                                `validate:"required,endsnotwith=/"`
}

var (
	log          *logrus.Entry
	config       Config
	mailAccounts map[string]*mail.SMTPServer   = make(map[string]*mail.SMTPServer)
	forms        map[string]*form_service.Form = make(map[string]*form_service.Form)
)

func main() {
	log = logrus.WithFields(logrus.Fields{
		"service": "POSTalion",
	})
	log.Info("starting up service")

	initConfig()

	for name, account := range config.MailAccounts {
		server, err := account.Init()
		if err != nil {
			log.Fatal(err)
		}
		mailAccounts[name] = server
	}
	if len(mailAccounts) == 0 {
		log.Error("No mail accounts defined, exiting...")
		return
	}

	for name, form := range config.Forms {
		form.Template = fmt.Sprintf("%s/%s", config.TemplateDir, form.Template)
		forms[name] = form
	}
	if len(forms) == 0 {
		log.Error("No forms defined, exiting...")
		return
	}

	router := pat.New()

	router.Post("/form/{name}", formHandler)

	http.Handle("/", router)

	log.Infof("listening on port %v", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", config.Port), nil))
}

func initConfig() {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("postalion")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/postalion")
	v.AddConfigPath("/data")
	v.SetEnvPrefix("postalion")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Infoln("config file not found, falling back to evironment variables")
		} else {
			log.Fatalf("error loading configuration: %v", err)
		}
	}

	err := v.Unmarshal(&config)
	if err != nil {
		log.Fatal(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		log.Fatal(err)
	}
}

func formHandler(wr http.ResponseWriter, req *http.Request) {
	name := req.URL.Query().Get(":name")
	form := forms[name]
	if form == nil {
		wr.WriteHeader(http.StatusNotFound)
		wr.Write([]byte(fmt.Sprintf("A form with name %s does not exist", name)))
		return
	} else {
		log := log.WithField("form", name)
		if form.Honeypot != "" && req.FormValue(form.Honeypot) != "" {
			wr.WriteHeader(http.StatusOK)
			log.Infof("received bad submission (honeypot = '%s'), ignoring...", req.FormValue(form.Honeypot))
			return
		}
		formContent, formFiles, err := form.Parse(req)
		if err != nil {
			log.Error(err)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		account := mailAccounts[form.Account]
		client, err := account.Connect()
		if err != nil {
			log.Error(err)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		body, err := form.HTML(formContent)
		if err != nil {
			log.Error(err)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		message := mail.NewMSG().
			SetFrom(form.From).
			AddTo(form.To).
			SetSubject(form.GetSubject(req)).
			SetBody(mail.TextHTML, body)

		if replyTo := form.GetReplyTo(req); replyTo != "" {
			message.SetReplyTo(replyTo)
		}

		for _, file := range formFiles {
			message.Attach(file)
		}

		if message.Error != nil {
			log.Error(message.Error)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := message.Send(client); err != nil {
			log.Error(err)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	wr.Header().Add("Location", form.Redirect)
	wr.WriteHeader(http.StatusSeeOther)
}
