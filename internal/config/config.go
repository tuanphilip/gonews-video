package config

type Config struct {
	NineRouterURL   string `json:"nineRouterUrl"`
	NineRouterKey   string `json:"nineRouterKey"`
	NineRouterModel string `json:"nineRouterModel"`
	OutputDir       string `json:"outputDir"`
}

func DefaultConfig() Config {
	return Config{
		NineRouterURL:   "https://9router.cool.khokey.com/v1",
		NineRouterKey:   "sk-f03f87025df45d45-7jf7y5-1e715a86",
		NineRouterModel: "ag/gemini-3.8-flash-medium",
		OutputDir:       "./output",
	}
}
