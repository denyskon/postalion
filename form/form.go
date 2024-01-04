// SPDX-License-Identifier: MIT
// Copyright 2023 Denys Konovalov <kontakt@denyskon.de>

package form

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"slices"

	mail "github.com/xhit/go-simple-mail/v2"
)

type Form struct {
	Account      string
	From         string `validate:"required"`
	To           string `validate:"required"`
	Honeypot     string
	ReplyTo      *ReplyTo
	Subject      string `validate:"required_without=SubjectField"`
	SubjectField string
	Template     string  `validate:"required,filepath"`
	Redirect     string  `validate:"required,http_url"`
	Fields       []Field `validate:"required"`
}

type Field struct {
	Name  string `validate:"required"`
	Label string `validate:"required"`
	Type  string `validate:"required,oneof=string file"`
}

type ReplyTo struct {
	Format string   `validate:"required"`
	Fields []string `validate:"required"`
}

type StringField struct {
	Label   string
	Content string
}

func (f Form) Parse(req *http.Request) (map[string]*StringField, []*mail.File, error) {
	fieldsToParse := slices.DeleteFunc[[]Field, Field](f.Fields, func(field Field) bool {
		return f.SubjectField == field.Name
	})

	formContent := make(map[string]*StringField)
	var formFiles []*mail.File

	for _, field := range fieldsToParse {
		switch field.Type {
		case "string":
			{

				formContent[field.Name] = &StringField{
					Label:   field.Label,
					Content: req.FormValue(field.Name),
				}
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

func (f Form) GetReplyTo(req *http.Request) string {
	if f.ReplyTo != nil {
		var fields []any
		for _, fieldName := range f.ReplyTo.Fields {
			fields = append(fields, req.FormValue(fieldName))
		}
		return fmt.Sprintf(f.ReplyTo.Format, fields...)
	}

	return ""
}

func (f Form) GetSubject(req *http.Request) string {
	if f.SubjectField != "" {
		return req.FormValue(f.SubjectField)
	}

	return f.Subject
}

func (f Form) HTML(content map[string]*StringField) (string, error) {
	tmpl, err := template.ParseFiles(f.Template)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, content)

	return b.String(), err
}
