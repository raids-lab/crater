package api

import (
	"net/http"
	"strings"

	"github.com/imroc/req/v3"
)

const maxPlainTextErrorRunes = 512

type responseStatus interface {
	GetStatusCode() int
	IsSuccessState() bool
}

type responseDetails interface {
	responseStatus
	GetContentType() string
	String() string
}

// errorFromRequest keeps HTTP failures distinct from transport and decode failures.
func errorFromRequest(resp *req.Response, requestErr error, craterCode int, msg string) error {
	if resp != nil && resp.GetStatusCode() != 0 {
		if err := errorFromResponse(resp, craterCode, msg); err != nil {
			return err
		}
		if requestErr != nil {
			return &RequestError{HTTPStatus: resp.GetStatusCode(), Msg: "platform returned an invalid response"}
		}
	}
	if requestErr != nil {
		return &NetworkError{Cause: requestErr}
	}
	return nil
}

func errorFromResponse(resp responseStatus, craterCode int, msg string) error {
	status := resp.GetStatusCode()
	if !resp.IsSuccessState() {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        responseErrorMessage(resp, msg),
		}
	}
	if craterCode != 0 {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        msg,
		}
	}
	return nil
}

func responseErrorMessage(resp responseStatus, msg string) string {
	if msg = strings.TrimSpace(msg); msg != "" {
		return msg
	}
	if detailed, ok := resp.(responseDetails); ok && strings.HasPrefix(strings.ToLower(detailed.GetContentType()), "text/plain") {
		plainText := strings.Join(strings.Fields(detailed.String()), " ")
		if runes := []rune(plainText); len(runes) > maxPlainTextErrorRunes {
			plainText = string(runes[:maxPlainTextErrorRunes]) + "..."
		}
		if plainText != "" {
			return plainText
		}
	}
	return http.StatusText(resp.GetStatusCode())
}
