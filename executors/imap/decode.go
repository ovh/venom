package imap

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/yesnault/go-imap/imap"
)

func decodeHeader(msg *mail.Message, headerName string) (string, error) {
	dec := new(mime.WordDecoder)
	s, err := dec.DecodeHeader(msg.Header.Get(headerName))
	if err != nil {
		return msg.Header.Get(headerName), fmt.Errorf("Error while decode header %s:%s", headerName, msg.Header.Get(headerName))
	}
	return s, nil
}

func hash(in string) string {
	h2 := md5.New()
	io.WriteString(h2, in)
	return fmt.Sprintf("%x", h2.Sum(nil))
}

func extract(rsp imap.Response, l *log.Entry) (*Mail, error) {
	tm := &Mail{}
	var params map[string]string

	header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
	body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.TEXT"])
	if mmsg, _ := mail.ReadMessage(bytes.NewReader(header)); mmsg != nil {
		var errds error
		tm.Subject, errds = decodeHeader(mmsg, "Subject")
		if errds != nil {
			return nil, errds
		}
		l.Debugf("|-- subject computed %s", tm.Subject)
		var errdf error
		tm.From, errdf = decodeHeader(mmsg, "From")
		if errdf != nil {
			return nil, errdf
		}
		l.Debugf("|-- from %s", tm.From)

		var errpm error
		_, params, errpm = mime.ParseMediaType(mmsg.Header.Get("Content-Type"))
		if errpm != nil {
			return nil, fmt.Errorf("Error while read Content-Type:%s", errpm)
		}

		date := strings.Replace(mmsg.Header.Get("Date"), "(UTC)", "", 1)
		idx := strings.Index(date, "(") // remove (TS)
		if idx > 0 {
			date = date[0:idx]
		}
		date = strings.Trim(date, " ")
		t, errt := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", date)
		if errt == nil {
			tm.Date = t
		} else {
			t2, errt2 := time.Parse("Fri, 17 Feb 2017 04:07:29 +0000 (UTC)", date)
			if errt2 == nil {
				tm.Date = t2
			} else {
				l.Errorf("Error while converting date:%s", errt2.Error())
				tm.Date = time.Now()
			}
		}
	}

	r := quotedprintable.NewReader(bytes.NewReader(body))
	bodya, errra := ioutil.ReadAll(r)
	if errra == nil {
		l.Debugf("Decode quotedprintable OK")
		tm.Body = string(bodya)
		return tm, nil
	} else if len(params) > 0 {
		l.Debugf("Error while read body:%s", errra.Error())
		r := bytes.NewReader(body)
		mr := multipart.NewReader(r, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				l.Debugf("errA:%s", err)
				continue
			}
			if err != nil {
				l.Debugf("--------> errB:%s", err)
				break
			}
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				l.Debugf("errC:%s", err)
				continue
			}
			l.Infof("Decode slurp OK")
			tm.Body = string(slurp)
			break
		}
		l.Debugf("stepD")
	}

	if tm.Body == "" {
		l.Debugf("EmptyBody, take body")
		tm.Body = string(bodya)
	} else {
		l.Debugf("stepF, tm.Body is ok")
	}
	return tm, nil
}
