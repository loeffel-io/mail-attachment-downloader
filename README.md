# Mail Downloader

[![Go Report Card](https://goreportcard.com/badge/github.com/loeffel-io/mail-downloader)](https://goreportcard.com/report/github.com/loeffel-io/mail-downloader)

Download attachments and mails as pdf through useful filters.
Years later, this tool is still a thing. I use it every month to download all my invoices from my mail accounts since years.

<img src="https://raw.githubusercontent.com/loeffel-io/mail-downloader/master/preview.gif" alt="preview" width="800">

### Usage

```bash
make build
./mail-downloader -config=config.yml -from="2019-10-01" -to="2019-12-31"
```

### Config

```yaml
imap:
  username: secret@gmail.com
  password: secret
  server: imap.gmail.com
  port: 993

attachments:
  mimetypes:
    - application/pdf

mails:
  subjects: # subject contains
    - invoice, amazon # invoice AND amazon
    - rechnung # OR rechnung
    - receipt # OR receipt
```

### Output

```text
files
├── secret@gmail.com
    ├── December-2019
    │   ├── marketplace.amazon.de
    │   │   │── invoice.pdf
    │   ├── iconfinder.com
    │       │── invoice.pdf
    │       │── invoice-2.pdf
    │       │── mail-123.pdf
    │
    ├── November -2019
        ├── facebook.com
            │── invoice.pdf
```
