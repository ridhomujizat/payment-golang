package enum

type HTTPMethodEnum string

const (
	GET    HTTPMethodEnum = "GET"
	POST   HTTPMethodEnum = "POST"
	PUT    HTTPMethodEnum = "PUT"
	DELETE HTTPMethodEnum = "DELETE"
)

func (e HTTPMethodEnum) ToString() string {
	switch e {
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case DELETE:
		return "DELETE"
	default:
		return ""
	}
}
