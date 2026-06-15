package api

type responseStatus interface {
	GetStatusCode() int
	IsSuccessState() bool
}

func errorFromResponse(resp responseStatus, craterCode int, msg string) error {
	status := resp.GetStatusCode()
	if !resp.IsSuccessState() {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        msg,
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
