package session

// Session holds all states for a single client connection.
type Session struct {
	SelectedDB int
}

func New() *Session {
	return &Session{
		SelectedDB: 0, // Default to DB 0
	}
}
