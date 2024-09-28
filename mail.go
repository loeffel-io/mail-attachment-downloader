package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	i "github.com/emersion/go-imap"
	m "github.com/emersion/go-message/mail"
	"github.com/gabriel-vasile/mimetype"
	"github.com/loeffel-io/mail-downloader/counter"
)

type mail struct {
	Uid         uint32
	MessageID   string
	Subject     string
	From        []*i.Address
	Date        time.Time
	Body        [][]byte
	Attachments []*attachment
	Error       error
}

type attachment struct {
	Filename string
	Body     []byte
	Mimetype string
}

func (mail *mail) fetchMeta(message *i.Message) {
	mail.Uid = message.Uid
	mail.MessageID = message.Envelope.MessageId
	mail.Subject = message.Envelope.Subject
	mail.From = message.Envelope.From
	mail.Date = message.Envelope.Date
}

func (mail *mail) fetchBody(reader *m.Reader) error {
	var (
		bodies      [][]byte
		attachments []*attachment
		count       = counter.CreateCounter()
	)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF || err.Error() == "multipart: NextPart: EOF" {
				break
			}

			return err
		}

		switch header := part.Header.(type) {
		case *m.InlineHeader:
			body, err := io.ReadAll(part.Body)
			if err != nil {
				if err == io.ErrUnexpectedEOF {
					continue
				}

				return err
			}

			bodies = append(bodies, body)
		case *m.AttachmentHeader:
			// This is an attachment
			filename, err := header.Filename()
			if err != nil {
				return err
			}

			body, err := io.ReadAll(part.Body)
			if err != nil {
				return err
			}

			mime := mimetype.Detect(body)

			if filename == "" {
				filename = fmt.Sprintf("%d-%d%s", mail.Uid, count.Next(), mime.Extension())
			}

			filename = new(imap).fixUtf(filename)

			// Replace all slashes with dashes to prevent directory traversal
			filename = strings.ReplaceAll(filename, "/", "-")

			attachments = append(attachments, &attachment{
				Filename: filename,
				Body:     body,
				Mimetype: mime.String(),
			})
		}
	}

	mail.Body = bodies
	mail.Attachments = attachments

	return nil
}

func (mail *mail) generatePdf() ([]byte, error) {
	count := counter.CreateCounter()

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	pdfg.LowQuality.Set(true)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)

	for _, body := range mail.Body {
		if mime := mimetype.Detect(body); !mime.Is("text/html") {
			continue
		}

		page := wkhtmltopdf.NewPageReader(bytes.NewReader(body))
		page.DisableJavascript.Set(true)
		page.Encoding.Set("UTF-8")

		pdfg.AddPage(page)
		count.Next()
	}

	if count.Current() == 0 {
		return nil, nil
	}

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	return pdfg.Bytes(), nil
}

func (mail *mail) getDirectoryName(username string) string {
	return fmt.Sprintf(
		"files/%s/%s-%d/%s",
		username, mail.Date.Month(), mail.Date.Year(), mail.From[0].HostName,
	)
}

func (mail *mail) getErrorText() string {
	return fmt.Sprintf("Error: %s\nSubject: %s\nFrom: %s\n", mail.Error.Error(), mail.Subject, mail.Date)
}
