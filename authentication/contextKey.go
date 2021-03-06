package authentication

type contextKey string

func (c contextKey) String() string {
	return string(c)
}

var (
	clientId = contextKey("client-id")
)
