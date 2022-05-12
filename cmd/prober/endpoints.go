package main

var (
	GET  = "GET"
	POST = "POST"
)

type ReadProberCheck struct {
	endpoint string
	method   string
	body     string
}

/*
/api/v1/index/retrieve
/api/v1/log
/api/v1/log/publicKey
/api/v1/log/proof
/api/v1/log/entries
/api/v1/log/entries
/api/v1/log/entries/retrieve
*/

var RekorEndpoints = []ReadProberCheck{
	{
		endpoint: "/api/v1/version",
		method:   GET,
	},
	{
		endpoint: "/api/v1/log/publicKey",
		method:   GET,
	},
}
