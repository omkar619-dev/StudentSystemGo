package middlewares

type HPPOptions struct {
	// Add any options you want to configure for the HPP middleware
	CheckQuery bool
	CheckBody  bool
	CheckBodyOnlyForContentType string
	Whitelist []string
}