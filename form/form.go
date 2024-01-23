// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package form

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

type Form struct {
	Type          string `validate:"required,oneof=mail carddav"`
	Honeypot      string
	Redirect      string         `validate:"required,http_url"`
	CardDavConfig *CardDavConfig `validate:"required_if=Type carddav"`
	MailConfig    *MailConfig    `validate:"required_if=Type mail"`
}

type CardDavConfig struct {
	DavEndpoint string                     `validate:"required"`
	ContactPath *FormattedField            `validate:"required"`
	Username    string                     `validate:"required"`
	Password    string                     `validate:"required"`
	Fields      map[string]*FormattedField `validate:"required"`
}

type MailConfig struct {
	Account      string `validate:"required"`
	From         string `validate:"required"`
	To           string `validate:"required"`
	Subject      string `validate:"required_without=SubjectField"`
	SubjectField string
	ReplyTo      *FormattedField
	Template     string  `validate:"required_if=Type mail"`
	Fields       []Field `validate:"required"`
}

type FormattedField struct {
	Format string   `validate:"required"`
	Fields []string `validate:"required"`
}

type Field struct {
	Name  string `validate:"required"`
	Label string `validate:"required"`
	Type  string `validate:"required,oneof=string file"`
}

type StringField struct {
	Name    string
	Label   string
	Content string
}

func (f FormattedField) GetFormattedString(req *http.Request) string {
	var fields []any
	for _, fieldName := range f.Fields {
		fields = append(fields, req.FormValue(fieldName))
	}
	return fmt.Sprintf(f.Format, fields...)
}

func (f Form) HTML(content []*StringField) (string, error) {
	tmpl, err := template.ParseFiles(f.MailConfig.Template)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, content)

	return b.String(), err
}
