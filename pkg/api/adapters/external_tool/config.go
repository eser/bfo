package external_tool

type Config struct {
	URL    string `conf:"URL"    default:"https://external-tool.dev/v1/tools/execute"`
	APIKey string `conf:"APIKEY"`
}
