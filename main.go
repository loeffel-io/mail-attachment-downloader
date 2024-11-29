package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/loeffel-io/mail-downloader/search"
	"github.com/luabagg/orcgen/v2"
	"gopkg.in/yaml.v3"
)

type PdfError struct {
	From    string
	Date    time.Time
	Subject string
	Err     error
}

func main() {
	var config *Config

	// flags
	configPath := flag.String("config", "", "config path")
	from := flag.String("from", "", "from date")
	to := flag.String("to", "", "to date")
	flag.Parse()

	// yaml
	yamlBytes, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	// yaml to config
	err = yaml.Unmarshal(yamlBytes, &config)
	if err != nil {
		log.Fatal(err)
	}

	// imap
	imap := &imap{
		Username: config.Imap.Username,
		Password: config.Imap.Password,
		Server:   config.Imap.Server,
		Port:     config.Imap.Port,
	}

	if err := imap.connect(); err != nil {
		log.Fatal(err)
	}

	if err := imap.login(); err != nil {
		log.Fatal(err)
	}

	imap.enableCharsetReader()

	// Mailbox
	_, err = imap.selectMailbox("INBOX")
	if err != nil {
		log.Fatal(err)
	}

	// search uids
	fromDate, err := time.Parse("2006-01-02", *from) // yyyy-MM-dd ISO 8601
	if err != nil {
		log.Fatal(err)
	}

	toDate, err := time.Parse("2006-01-02", *to) // yyyy-MM-dd ISO 8601
	if err != nil {
		log.Fatal(err)
	}

	pdfGen := orcgen.NewHandler(orcgen.PDFConfig{
		Landscape:         false,
		PrintBackground:   true,
		PreferCSSPageSize: true,
	})
	pdfGen.SetFullPage(true)

	uids, err := imap.search(fromDate, toDate)
	if err != nil {
		log.Fatal(err)
	}

	// seqset
	seqset := imap.createSeqSet(uids)

	// channel
	mailsChan := make(chan *mail)

	// fetch messages
	go func() {
		if err = imap.fetchMessages(seqset, mailsChan); err != nil {
			log.Fatal(err)
		}

		close(mailsChan)
	}()

	// start bar
	fmt.Printf("%s: fetching messages...\n", imap.Username)
	bar := pb.StartNew(len(uids))

	// mails
	mails := make([]*mail, 0)

	// fetch messages
	for mail := range mailsChan {
		mails = append(mails, mail)
		bar.Increment()
	}

	// logout
	if err := imap.Client.Logout(); err != nil {
		log.Fatal(err)
	}

	// start bar
	fmt.Printf("%s: processing messages...\n", imap.Username)
	bar.SetCurrent(0)

	// process messages
	var pdfErrors []*PdfError
	for _, mail := range mails {
		dir := mail.getDirectoryName(imap.Username)

		if mail.Error != nil {
			log.Println(mail.getErrorText())
			bar.Increment()
			continue
		}

		// attachments
		for _, attachment := range mail.Attachments {
			s := &search.Search{
				Search: config.Attachments.Mimetypes,
				Data:   attachment.Mimetype,
			}

			if !s.Find() {
				continue
			}

			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				log.Fatal(err)
			}

			if err = os.WriteFile(fmt.Sprintf("%s/%s", dir, attachment.Filename), attachment.Body, 0o644); err != nil {
				log.Printf("attachment.Filename: %s", attachment.Filename)
				if pe, ok := err.(*os.PathError); ok {
					if pe.Err == syscall.ENAMETOOLONG {
						log.Println(err.Error())
						continue
					}
				}
				log.Fatal(err)
			}
		}

		// pdf
		s := &search.Search{
			Search: config.Mails.Subjects,
			Data:   mail.Subject,
		}

		if !s.Find() {
			bar.Increment()
			continue
		}

		fileInfo, err := mail.generatePdf(pdfGen)
		if err != nil {
			switch err {
			case ErrNoHtmlBody:
				pdfErrors = append(pdfErrors, &PdfError{
					From:    mail.From[0].Address(),
					Date:    mail.Date,
					Subject: mail.Subject,
					Err:     err,
				})
				bar.Increment()
				continue
			default:
				pdfErrors = append(pdfErrors, &PdfError{
					From:    mail.From[0].Address(),
					Date:    mail.Date,
					Subject: mail.Subject,
					Err:     err,
				})
				bar.Increment()
				continue
			}
		}

		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Fatal(err)
		}

		if err := os.WriteFile(fmt.Sprintf("%s/mail-%d.pdf", dir, mail.Uid), fileInfo.File, 0o644); err != nil {
			log.Fatal(err)
		}

		bar.Increment()
	}

	if len(pdfErrors) > 0 {
		fmt.Printf("%s: found some errors\n", imap.Username)

		for _, pdfError := range pdfErrors {
			fmt.Printf("%s - %s - %s - %s\n", pdfError.From, pdfError.Date, pdfError.Subject, pdfError.Err.Error())
		}
	}

	// done
	fmt.Println("Done")
}
