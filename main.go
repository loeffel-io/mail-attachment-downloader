package main

import (
	"flag"
	"fmt"
	"github.com/cheggaaa/pb"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func main() {
	// flags
	username := flag.String("username", "", "username")
	password := flag.String("password", "", "password")
	server := flag.String("server", "", "server")
	port := flag.String("port", "", "port")
	from := flag.String("from", "", "from")
	to := flag.String("to", "", "to")
	flag.Parse()

	// imap
	imap := &imap{
		Username: *username,
		Password: *password,
		Server:   *server,
		Port:     *port,
	}

	if err := imap.connect(); err != nil {
		log.Fatal(err)
	}

	if err := imap.login(); err != nil {
		log.Fatal(err)
	}

	imap.enableCharsetReader()

	// Mailbox
	_, err := imap.selectMailbox("INBOX")

	// search uids
	fromDate, err := time.Parse("2006-01-02", *from) // yyyy-MM-dd ISO 8601

	if err != nil {
		log.Fatal(err)
	}

	toDate, err := time.Parse("2006-01-02", *to) // yyyy-MM-dd ISO 8601

	if err != nil {
		log.Fatal(err)
	}

	uids, err := imap.search(fromDate, toDate)

	if err != nil {
		log.Fatal(err)
	}

	// seqset
	seqset := imap.createSeqSet(uids)

	// channel
	var mailsChan = make(chan *mail)

	// start bar
	bar := pb.StartNew(len(uids))

	// fetch messages
	go func() {
		if err = imap.fetchMessages(seqset, mailsChan); err != nil {
			log.Fatal(err)
		}
	}()

	// out messages
	for mail := range mailsChan {
		dir := mail.getDirectoryName(imap.Username)

		if mail.Error != nil {
			log.Println(mail.getErrorText())
			bar.Increment()
			continue
		}

		// create dir
		if len(mail.Attachments) != 0 || len(mail.Body) != 0 {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				log.Fatal(err)
			}
		}

		// attachments
		for _, attachment := range mail.Attachments {
			if err = ioutil.WriteFile(fmt.Sprintf("%s/%s", dir, attachment.Filename), attachment.Body, 0644); err != nil {
				log.Fatal(err)
			}
		}

		// pdf
		bytes, err := mail.generatePdf()

		if err != nil {
			log.Fatal(err)
		}

		if err = ioutil.WriteFile(fmt.Sprintf("%s/mail-%d.pdf", dir, mail.Uid), bytes, 0644); err != nil {
			log.Fatal(err)
		}

		bar.Increment()
	}
}
