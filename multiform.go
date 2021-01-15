package multiform

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"sort"
	"strings"
)

type FormBuilder struct {
	Parts    []FormBuilderPart
	reader   io.Reader
	boundary string
}

type FormBuilderPart struct {
	Header textproto.MIMEHeader
	Body   io.Reader
}

func NewBuilder() FormBuilder {
	return FormBuilder{
		boundary: randomBoundary(),
	}
}

func (f *FormBuilder) AddPart(header textproto.MIMEHeader, body io.Reader) {
	f.Parts = append(f.Parts, FormBuilderPart{Header: header, Body: body})
}

func (f *FormBuilder) AddFormField(fieldName string, body io.Reader) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldName)))
	f.AddPart(h, body)
}

func (f *FormBuilder) AddFormFile(fieldName string, fileName string, body io.ReadCloser) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(fieldName), escapeQuotes(fileName)))
	h.Set("Content-Type", "application/octet-stream")
	f.AddPart(h, body)
}

func (f *FormBuilder) AddField(fieldName string, fieldValue string) {
	f.AddFormField(fieldName, strings.NewReader(fieldValue))
}

func (f *FormBuilder) SetBoundary(boundary string) error {
	// rfc2046#section-5.1.1
	if len(boundary) < 1 || len(boundary) > 70 {
		return errors.New("mime: invalid boundary length")
	}
	end := len(boundary) - 1
	for i, b := range boundary {
		if 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9' {
			continue
		}
		switch b {
		case '\'', '(', ')', '+', '_', ',', '-', '.', '/', ':', '=', '?':
			continue
		case ' ':
			if i != end {
				continue
			}
		}
		return errors.New("mime: invalid boundary character")
	}
	f.boundary = boundary
	return nil
}

func (f *FormBuilder) FormDataContentType() string {
	b := f.boundary
	// We must quote the boundary if it contains any of the
	// tspecials characters defined by RFC 2045, or space.
	if strings.ContainsAny(b, `()<>@,;:\"/[]?= `) {
		b = `"` + b + `"`
	}
	return "multipart/form-data; boundary=" + b
}

func (f *FormBuilder) Done() {
	if f.reader == nil {
		f.makeMultiReader()
	}
}

func (f FormBuilder) Close() (err error) {
	for _, part := range f.Parts {
		if err = closeReader(part.Body); err != nil {
			return
		}
	}
	return
}

func (f FormBuilder) Read(b []byte) (int, error) {
	if f.reader == nil {
		return 0, errors.New("Must end the builder before reading")
	}
	return f.reader.Read(b)
}

func (f *FormBuilder) makeMultiReader() {
	parts := []io.Reader{}
	for i, part := range f.Parts {
		isFirstPart := i == 0
		header := makePartHeader(part.Header, f.boundary, isFirstPart)
		// TODO: Should we throw error in read method if the boundary appears in the
		// body somewhere or just let the client detect the encoding error?
		parts = append(parts, header, part.Body)
	}
	parts = append(parts, makePartFooter(f.boundary))
	f.reader = io.MultiReader(parts...)
}

func closeReader(r interface{}) error {
	if r == nil {
		return nil
	}
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

func makePartFooter(boundary string) io.Reader {
	return strings.NewReader(fmt.Sprintf("\r\n--%s--\r\n", boundary))
}

func makePartHeader(header textproto.MIMEHeader, boundary string, isFirst bool) io.Reader {
	var b bytes.Buffer
	if isFirst {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
	} else {
		fmt.Fprintf(&b, "\r\n--%s\r\n", boundary)
	}
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range header[k] {
			fmt.Fprintf(&b, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(&b, "\r\n")
	return &b
}
