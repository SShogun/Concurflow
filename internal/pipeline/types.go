package pipeline

type RawURL struct {
	URL string
	ID  int
}

type Reason string

const (
	fair              Reason = "fair"
	empty             Reason = "empty"
	malformedURL      Reason = "malformed_url"
	missingScheme     Reason = "missing_scheme"
	missingHost       Reason = "missing_host"
	unsupportedScheme Reason = "unsupported_scheme"
)

type NormalizedURL struct {
	ID     int
	URL    string
	Valid  bool
	Reason []Reason
}
