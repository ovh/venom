package imap

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"github.com/ovh/venom"
	"github.com/yesnault/go-imap/imap"
)

func decodeHeader(msg *mail.Message, headerName string) (string, error) {
	dec := new(mime.WordDecoder)
	s, err := dec.DecodeHeader(msg.Header.Get(headerName))
	if err != nil {
		return msg.Header.Get(headerName), err
	}
	return s, nil
}

// extract tries to read the content of the mail contained in rsp
func extract(ctx context.Context, rsp imap.Response) (Mail, error) {
	var (
		m   = Mail{}
		msg *mail.Message
		err error
	)

	if uidAttr, ok := rsp.MessageInfo().Attrs["UID"]; ok {
		m.UID = imap.AsNumber(uidAttr)
	}

	if headerAttr, ok := rsp.MessageInfo().Attrs["RFC822.HEADER"]; ok {
		header := imap.AsBytes(headerAttr)
		msg, err = mail.ReadMessage(bytes.NewReader(header))
		if err != nil {
			return Mail{}, err
		}
		m.Subject, err = decodeHeader(msg, "Subject")
		if err != nil {
			return Mail{}, fmt.Errorf("Cannot decode Subject header: %s", err)
		}
		m.From, err = decodeHeader(msg, "From")
		if err != nil {
			return Mail{}, fmt.Errorf("Cannot decode From header: %s", err)
		}
		m.To, err = decodeHeader(msg, "To")
		if err != nil {
			return Mail{}, fmt.Errorf("Cannot decode To header: %s", err)
		}
	}

	if textAttr, ok := rsp.MessageInfo().Attrs["RFC822.TEXT"]; ok {
		body := imap.AsBytes(textAttr)
		encoding := msg.Header.Get("Content-Transfer-Encoding")
		var r io.Reader = bytes.NewReader(body)
		switch encoding {
		case "7bit", "8bit", "binary":
			// noop, reader already initialized.
		case "quoted-printable":
			r = quotedprintable.NewReader(r)
		case "base64":
			r = base64.NewDecoder(base64.StdEncoding, r)
		}
		venom.Debug(ctx, "Mail Content-Transfer-Encoding is %s ", encoding)

		contentType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
		// Using strings.Contains because "mime: no media type" error is not defined in mime package
		// and thus, cannot be used with errors.Is
		if err != nil && !strings.Contains(err.Error(), "mime: no media type") {
			return Mail{}, fmt.Errorf("Error while reading Content-Type:%s", err)
		} else if err != nil && strings.Contains(err.Error(), "mime: no media type") {
			// Error "mime: no media type" is not a blocking error (returns when no Content-Type header is present)
			// But it means we cannot read body
			venom.Warn(ctx, "Content-Type header empty, skipping body reading")
		}
		if contentType != "" {
			if contentType == "multipart/mixed" || contentType == "multipart/alternative" {
				if boundary, ok := params["boundary"]; ok {
					mr := multipart.NewReader(r, boundary)
					for {
						p, errm := mr.NextPart()
						if errm == io.EOF {
							continue
						}
						if errm != nil {
							venom.Debug(ctx, "Error while read Part:%s", err)
							break
						}
						slurp, errm := io.ReadAll(p)
						if errm != nil {
							venom.Debug(ctx, "Error while ReadAll Part:%s", err)
							continue
						}
						slurp = bytes.TrimRight(slurp, "\r\n")
						m.Body = string(slurp)
						break
					}
				}
			} else {
				body, err = io.ReadAll(r)
				if err != nil {
					return Mail{}, err
				}
				body = bytes.TrimRight(body, "\r\n")
			}
			if m.Body == "" {
				body = bytes.TrimRight(body, "\r\n")
				m.Body = string(body)
			}
		}
	}

	if flagsAttr, ok := rsp.MessageInfo().Attrs["FLAGS"]; ok {
		flagSetStr := imap.AsFlagSet(flagsAttr).String()
		// Flags without embracing parenthesis
		m.Flags = strings.Split(flagSetStr[1:len(flagSetStr)-1], " ")
	}

	return m, nil
}
