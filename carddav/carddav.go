// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package carddav

import (
	"context"
	"net/http"

	forms_service "github.com/denyskon/postalion/form"
	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
)

func HandleForm(form *forms_service.Form, req *http.Request) error {
	config := form.CardDavConfig

	c, err := carddav.NewClient(
		webdav.HTTPClientWithBasicAuth(nil, config.Username, config.Password),
		config.DavEndpoint,
	)
	if err != nil {
		return err
	}

	card := vcard.Card{
		"VERSION": []*vcard.Field{{Value: "3.0"}},
	}

	for name, field := range config.Fields {
		card.SetValue(name, field.GetFormattedString(req))
	}

	path := config.ContactPath.GetFormattedString(req)
	ctx := context.Background()
	if _, err = c.PutAddressObject(ctx, path, card); err != nil {
		return err
	}

	return nil
}
