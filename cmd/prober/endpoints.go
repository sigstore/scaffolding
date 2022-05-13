package main

var (
	GET  = "GET"
	POST = "POST"
)

type ReadProberCheck struct {
	endpoint string
	method   string
	body     string
	queries  map[string]string
}

var RekorEndpoints = []ReadProberCheck{
	{
		endpoint: "/api/v1/version",
		method:   GET,
	}, {
		endpoint: "/api/v1/log/publicKey",
		method:   GET,
	},
	{
		endpoint: "/api/v1/log",
		method:   GET,
	}, {
		endpoint: "/api/v1/log/entries",
		method:   GET,
		queries:  map[string]string{"logIndex": "10"},
	}, {
		endpoint: "/api/v1/log/proof",
		method:   GET,
		queries:  map[string]string{"firstSize": "10", "lastSize": "20"},
	}, {
		endpoint: "/api/v1/log/entries/retrieve",
		method:   POST,
		body:     "{\"hash\":\"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b\"}",
	}, {
		endpoint: "/api/v1/index/retrieve",
		method:   POST,
		body:     "{\"hash\":\"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b\"}",
	},
}
