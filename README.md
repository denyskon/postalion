# 📃 POSTalion

Lightweight & painless HTML5 form backend

> Ever wanted to simply list a few fields, set an email account and receive emails every time someone submits your HTML5 form?
> Me too, and that's what POSTalion is for! While there are lots of different services for that usecase, none I tested were able
> to work independently to be used e. g. with a static site, could receive file uploads and send them as attachments and were
> well-maintained & up-to-date ([FormTools 😢](https://github.com/formtools/core)). That's why I created POSTalion.

⚠️ This application should not be considered stable and is provided without any guarantees

## Features

- accepting HTML5 POST forms
- sending emails using a SMTP server
  - Go HTML templates for emails
  - attachment support
  - pre-configure multiple email accounts
  - formatted replyTo and subject fields
- adding vCard contacts through a CardDav server

## Configuration

See `example.postalion.yml` for an example.

## Usage

Building:

`go build -o postalion .`

There is also a Docker image under `ghcr.io/denyskon/postalion:latest`
