package imap

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"

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

func extract(ctx context.Context, rsp imap.Response) (*Mail, error) {
	tm := &Mail{}

	header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
	tm.UID = imap.AsNumber((rsp.MessageInfo().Attrs["UID"]))
	body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.TEXT"])

	mmsg, err := mail.ReadMessage(bytes.NewReader(header))
	if err != nil {
		return nil, err
	}
	tm.Subject, err = decodeHeader(mmsg, "Subject")
	if err != nil {
		return nil, fmt.Errorf("Cannot decode Subject header: %s", err)
	}
	tm.From, err = decodeHeader(mmsg, "From")
	if err != nil {
		return nil, fmt.Errorf("Cannot decode From header: %s", err)
	}
	tm.To, err = decodeHeader(mmsg, "To")
	if err != nil {
		return nil, fmt.Errorf("Cannot decode To header: %s", err)
	}

	encoding := mmsg.Header.Get("Content-Transfer-Encoding")
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

	contentType, params, err := mime.ParseMediaType(mmsg.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("Error while reading Content-Type:%s", err)
	}
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
				slurp, errm := ioutil.ReadAll(p)
				if errm != nil {
					venom.Debug(ctx, "Error while ReadAll Part:%s", err)
					continue
				}
				tm.Body = string(slurp)
				break
			}
		}
	} else {
		body, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
	}
	if tm.Body == "" {
		tm.Body = string(body)
	}
	return tm, nil
}
