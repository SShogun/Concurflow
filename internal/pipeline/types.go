package pipeline

type RawURL struct {
	URL string
	ID  int
}

type Reason string

const (
	fair               Reason = "fair"
	empty              Reason = "empty"
	missing_scheme     Reason = "missing_scheme"
	unsupported_scheme Reason = "unsupported_scheme"
)

type NormalizedURL struct {
	ID     int
	URL    string
	Valid  bool
	Reason []Reason
}
