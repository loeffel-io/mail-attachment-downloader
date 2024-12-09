package mail

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	goimap "github.com/emersion/go-imap"
	gomessage "github.com/emersion/go-message/mail"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/luabagg/orcgen/v2"
	"github.com/luabagg/orcgen/v2/pkg/fileinfo"
	"github.com/luabagg/orcgen/v2/pkg/handlers"
)

var ErrNoHtmlBody = errors.New("no html body found")

type Mail struct {
	Uid         uint32
	MessageID   string
	Subject     string
	From        []*goimap.Address
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

func (mail *Mail) FetchMeta(message *goimap.Message) {
	mail.Uid = message.Uid
	mail.MessageID = message.Envelope.MessageId
	mail.Subject = message.Envelope.Subject
	mail.From = message.Envelope.From
	mail.Date = message.Envelope.Date
}

func (mail *Mail) FetchBody(reader *gomessage.Reader) error {
	var (
		bodies      [][]byte
		attachments []*attachment
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
		case *gomessage.InlineHeader:
			body, err := io.ReadAll(part.Body)
			if err != nil {
				if err == io.ErrUnexpectedEOF {
					continue
				}

				return err
			}

			bodies = append(bodies, body)
		case *gomessage.AttachmentHeader:
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
				filename = fmt.Sprintf("%d-%s%s", mail.Uid, uuid.New(), mime.Extension())
			}

			filename = fixUtf(filename)

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

func (mail *Mail) GeneratePdf(pdfGen handlers.FileHandler[orcgen.PDFConfig]) (*fileinfo.Fileinfo, error) {
	var htmlBody []byte
	var textBody []byte
	for _, body := range mail.Body {
		mime := mimetype.Detect(body)
		switch {
		case mime.Is("text/html"):
			htmlBody = append(htmlBody, body...)
		case mime.Is("text/plain"):
			textBody = append(textBody, body...)
		default:
			continue
		}
	}

	if len(htmlBody) != 0 {
		return orcgen.ConvertHTML(pdfGen, htmlBody)
	}

	if len(textBody) != 0 {
		return orcgen.ConvertHTML(pdfGen, textBody)
	}

	return nil, ErrNoHtmlBody
}

func (mail *Mail) GetDirectoryName(username string) string {
	return fmt.Sprintf(
		"files/%s/%s-%d/%s",
		username, mail.Date.Month(), mail.Date.Year(), mail.From[0].HostName,
	)
}

func (mail *Mail) GetErrorText() string {
	return fmt.Sprintf("Error: %s\nSubject: %s\nFrom: %s\n", mail.Error.Error(), mail.Subject, mail.Date)
}

func fixUtf(str string) string {
	callable := func(r rune) rune {
		if r == utf8.RuneError {
			return -1
		}

		return r
	}

	return strings.Map(callable, str)
}
