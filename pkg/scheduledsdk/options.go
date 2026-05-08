package scheduledsdk

import "net/http"

type Options struct {
	Adapter    Adapter
	BaseURL    string
	HTTPClient *http.Client
}
