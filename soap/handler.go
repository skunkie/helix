// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package soap

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/ethulhu/helix/logger"
)

var prefixRe = regexp.MustCompile(`<([a-zA-Z0-9_.-]+):([a-zA-Z0-9_.-]+)[\s/>]`)

// DetectPrefix extracts the XML namespace prefix used on the action element in a SOAP request body,
// or returns an empty string if no prefix was used.
func DetectPrefix(requestBody []byte, actionName string) string {
	for _, match := range prefixRe.FindAllSubmatch(requestBody, -1) {
		if string(match[2]) == actionName {
			return string(match[1])
		}
	}
	return ""
}

func Handle(w http.ResponseWriter, r *http.Request, handler Interface) {
	log, ctx := logger.FromContext(r.Context())

	soapAction := r.Header.Get("SOAPAction")
	if soapAction == "" {
		http.Error(w, "must set SOAPAction header", http.StatusBadRequest)
		log.Warning("missing SOAPAction header")
		return
	}

	parts := strings.Split(strings.Trim(soapAction, `"`), "#")
	if len(parts) != 2 {
		http.Error(w, fmt.Sprintf(`SOAPAction header must be of form "namespace#action", got %q`, soapAction), http.StatusBadRequest)
		log.WithField("soap.SOAPAction", soapAction).Warning("invalid SOAPAction header")
		return
	}

	namespace := parts[0]
	action := parts[1]
	log.AddField("soap.namespace", namespace)
	log.AddField("soap.action", action)

	envelope, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.WithError(err).Warning("could not read body of SOAP request")
		return
	}

	in, err := deserializeSOAPEnvelope(envelope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.WithError(err).Warning("could not deserialize SOAP request")
		return
	}

	out, err := handler.Call(ctx, namespace, action, in)

	w.Header().Set("Content-Type", `text/xml; charset="utf-8"`)
	w.Header().Set("Ext", "")

	if err != nil {
		var rErr Error
		if errors.As(err, &rErr) && rErr.FaultCode() != FaultServer {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	envelope = serializeSOAPEnvelope(out, err)
	envelope = bytes.ReplaceAll(envelope, []byte("&#34;"), []byte(`"`))
	_, _ = w.Write(envelope)

	if err != nil {
		log.WithError(err).Warning("served SOAP error")
		return
	}
	log.Debug("served SOAP request")
}
