package scheduledsdk

import "net/http"

type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	Adapter    Adapter
}
