package imap

import (
	"errors"
	"log"
	"time"

	goimap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	gomessage "github.com/emersion/go-message/mail"
	"github.com/loeffel-io/mail-downloader/internal/mail"
	"golang.org/x/text/encoding/charmap"
)

type Imap struct {
	Username string
	Password string
	Server   string
	Port     string
	Client   *client.Client
}

func (imap *Imap) Connect() error {
	c, err := client.DialTLS(imap.Server+":"+imap.Port, nil)
	if err != nil {
		return err
	}

	imap.Client = c
	return nil
}

func (imap *Imap) Login() error {
	return imap.Client.Login(imap.Username, imap.Password)
}

func (imap *Imap) SelectMailbox(mailbox string) (*goimap.MailboxStatus, error) {
	return imap.Client.Select(mailbox, true)
}

func (imap *Imap) Search(from, to time.Time) ([]uint32, error) {
	search := goimap.NewSearchCriteria()
	search.Since = from
	search.Before = to

	return imap.Client.UidSearch(search)
}

func (imap *Imap) CreateSeqSet(uids []uint32) *goimap.SeqSet {
	seqset := new(goimap.SeqSet)
	seqset.AddNum(uids...)

	return seqset
}

func (imap *Imap) EnableCharsetReader() {
	charset.RegisterEncoding("ansi", charmap.Windows1252)
	charset.RegisterEncoding("iso8859-15", charmap.ISO8859_15)
	goimap.CharsetReader = charset.Reader
}

func (imap *Imap) FetchMessages(seqset *goimap.SeqSet, mailsChan chan *mail.Mail) error {
	messages := make(chan *goimap.Message)
	section := new(goimap.BodySectionName)

	go func() {
		if err := imap.Client.UidFetch(seqset, []goimap.FetchItem{section.FetchItem(), goimap.FetchEnvelope}, messages); err != nil {
			log.Println(err)
		}
	}()

	for message := range messages {
		mail := new(mail.Mail)
		mail.FetchMeta(message)

		reader := message.GetBody(section)

		if reader == nil {
			return errors.New("no reader")
		}

		mailReader, err := gomessage.CreateReader(reader)
		if err != nil {
			mail.Error = err
			mailsChan <- mail

			if mailReader != nil {
				if err := mailReader.Close(); err != nil {
					log.Fatal(err)
				}
			}

			continue
		}

		mail.Error = mail.FetchBody(mailReader)
		mailsChan <- mail

		if mailReader != nil {
			if err := mailReader.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}

	return nil
}
