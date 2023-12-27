module github.com/jlevesy/workflows-exporter

go 1.21.4

require (
	github.com/gofri/go-github-ratelimit v1.0.5
	github.com/google/go-github/v57 v57.0.0
	github.com/migueleliasweb/go-github-mock v0.0.22
	github.com/prometheus/client_golang v1.18.0
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.26.0
	golang.org/x/oauth2 v0.15.0
	golang.org/x/sync v0.5.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-github/v56 v56.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Until https://github.com/gofri/go-github-ratelimit/pull/18 lands.
replace github.com/gofri/go-github-ratelimit => github.com/zendesk-piotrpawluk/go-github-ratelimit v0.0.0-20231120163947-01b70bdcdf9a
