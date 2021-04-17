package stalker

type Stream interface {
	GenerateLink() (string, error)
	GenerateRewrittenResponse(string) string
	CMD() string
}
