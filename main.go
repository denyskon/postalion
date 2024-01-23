// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package main

import (
	"fmt"
	"net/http"

	carddav_service "github.com/denyskon/postalion/carddav"
	form_service "github.com/denyskon/postalion/form"
	mail_service "github.com/denyskon/postalion/mail"
	"github.com/denyskon/postalion/version"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/pat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Config struct {
	MailAccounts map[string]*mail_service.MailAccount `validate:"dive,required"`
	Forms        map[string]*form_service.Form        `validate:"required,dive,required"`
	Port         int                                  `validate:"required"`
	TemplateDir  string                               `validate:"endsnotwith=/"`
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

	for name, form := range config.Forms {
		switch form.Type {
		case "mail":
			{
				if mailAccounts[form.MailConfig.Account] == nil {
					log.Fatalf("could not find mail account %s for form %s", form.MailConfig.Account, name)
				}
				form.MailConfig.Template = fmt.Sprintf("%s/%s", config.TemplateDir, form.MailConfig.Template)
			}
		case "carddav":
			{
			}
		default:
			{
				log.Fatalf("unknown type %s for form %s", form.Type, name)
			}
		}
		forms[name] = form
	}
	if len(forms) == 0 {
		log.Fatal("no forms defined, exiting...")
	}

	router := pat.New()

	router.Post("/form/{name}", formHandler)
	router.Get("/status", func(wr http.ResponseWriter, req *http.Request) {
		wr.WriteHeader(http.StatusOK)
		wr.Write([]byte(fmt.Sprintf("POSTalion %s", version.Version())))
	})

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
	v.AddConfigPath("/app")
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
		wr.Write([]byte(fmt.Sprintf("form with name %s does not exist", name)))
		return
	} else {
		log := log.WithField("form", name)
		if form.Honeypot != "" && req.FormValue(form.Honeypot) != "" {
			wr.WriteHeader(http.StatusOK)
			log.Infof("received bad submission (honeypot = '%s'), ignoring...", req.FormValue(form.Honeypot))
			return
		}

		var err error
		switch form.Type {
		case "mail":
			{
				acc := mailAccounts[form.MailConfig.Account]
				err = mail_service.HandleForm(form, req, acc)
			}
		case "carddav":
			{
				err = carddav_service.HandleForm(form, req)
			}
		}

		if err != nil {
			log.Error(err)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	wr.Header().Add("Location", form.Redirect)
	wr.WriteHeader(http.StatusSeeOther)
}
