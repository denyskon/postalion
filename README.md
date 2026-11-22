# 📃 POSTalion

Lightweight & painless HTML5 form backend

> Ever wanted to simply list a few fields, set an email account and receive emails every time someone submits your HTML5 form?
> Me too, and that's what POSTalion is for! While there are lots of different services for that usecase, none I tested were able
> to work independently to be used e. g. with a static site, could receive file uploads and send them as attachments and were
> well-maintained & up-to-date ([FormTools was my tool of choice before becoming unmaintained](https://github.com/formtools/core)). That's why I created POSTalion.

## Features

- accepting HTML5 POST forms
- sending emails using a SMTP server
  - Go HTML templates for emails
  - attachment support
  - pre-configure multiple email accounts
  - formatted replyTo and subject fields
- adding vCard contacts through a CardDav server
- optional spam checks using a local Ollama model

## Configuration

See `example.postalion.yml` for an example.

### Spam check

Any form can be checked for spam using an [Ollama](https://ollama.com) endpoint.
Add a `spamCheck` block to the form:

```yaml
forms:
  test-form:
    spamCheck:
      endpoint: http://localhost:11434
      model: llama3.2
      # optional: send users to a custom page instead of the default error page
      rejectRedirect: https://example.com/pages/message-not-sent
```

`endpoint` is the base URL of the Ollama server and `model` is the model used for
classification. If the Ollama endpoint cannot be reached, the submission is
treated as legitimate and processed normally.

When a submission is classified as spam, it is not processed. The user is shown
an error telling them the submission was not accepted, either through a
`422 Unprocessable Entity` response or by being redirected to `rejectRedirect`
if configured. This also lets legitimate users know when the spam filter made a
mistake, so they can contact you through another channel.

The configured endpoint and model are checked at startup, and a warning is
logged if the endpoint is unreachable or the model is missing. Structured
outputs require Ollama 0.5.0 or newer; on older versions the spam check fails
and submissions are processed normally.

## Usage

Building:

`go build -o postalion .`

There is also a Docker image under `ghcr.io/denyskon/postalion:latest`. Dev builds can be
published manually from the GitHub Actions workflow and are tagged
`ghcr.io/denyskon/postalion:dev`
