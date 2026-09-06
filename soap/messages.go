// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package soap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

type (
	envelope struct {
		XMLName  xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
		Encoding string   `xml:"encodingStyle,attr"`
		Header   header   `xml:"Header,omitempty"`
		Body     body     `xml:"Body,omitempty"`
	}

	header struct {
		XMLName  xml.Name `xml:"Header"`
		Contents []byte   `xml:",innerxml"`
	}
	body struct {
		XMLName  xml.Name `xml:"Body"`
		Fault    *fault   `xml:"Fault,omitempty"`
		Contents []byte   `xml:",innerxml"`
	}
	fault struct {
		XMLName xml.Name `xml:"Fault"`
		Code    string   `xml:"faultcode"`
		String  string   `xml:"faultstring"`
		Detail  struct {
			Contents string `xml:",innerxml"`
		} `xml:"detail"`
	}
)

// serializeSOAPEnvelope is kinda hacky because some devices don't like nested default namespaces.
func serializeSOAPEnvelope(body []byte, err error) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">`)
	buf.WriteString(`<s:Body>`)
	buf.Write(body)
	if err != nil {
		writeEscaped := func(value string) {
			_ = xml.EscapeText(&buf, []byte(value))
		}
		buf.WriteString(`<s:Fault>`)
		var rErr Error
		if errors.As(err, &rErr) {
			buf.WriteString(`<s:faultcode>s:`)
			writeEscaped(string(rErr.FaultCode()))
			buf.WriteString(`</s:faultcode><s:faultstring>`)
			writeEscaped(rErr.FaultString())
			buf.WriteString(`</s:faultstring><s:detail>`)
			buf.WriteString(rErr.Detail())
			buf.WriteString(`</s:detail>`)
		} else {
			fmt.Fprintf(&buf, `<s:faultcode>s:%v</s:faultcode>`, FaultServer)
			fmt.Fprintf(&buf, `<s:faultstring>Server Error</s:faultstring>`)
			buf.WriteString(`<s:detail>`)
			writeEscaped(err.Error())
			buf.WriteString(`</s:detail>`)
		}
		buf.WriteString(`</s:Fault>`)
	}
	buf.WriteString(`</s:Body>`)
	buf.WriteString(`</s:Envelope>`)
	return buf.Bytes()
}

func deserializeSOAPEnvelope(data []byte) ([]byte, error) {
	e := envelope{}
	if err := xml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("could not deserialize XML envelope: %w (%s)", err, data)
	}

	if e.Body.Fault != nil {
		faultCode := e.Body.Fault.Code
		if _, unqualified, ok := strings.Cut(faultCode, ":"); ok {
			faultCode = unqualified
		}
		return nil, remoteError{
			faultCode:   FaultCode(faultCode),
			faultString: e.Body.Fault.String,
			detail:      strings.TrimSpace(e.Body.Fault.Detail.Contents),
		}
	}
	return e.Body.Contents, nil
}
