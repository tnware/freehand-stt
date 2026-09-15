package compatibility

import "errors"

// NeMoOptions preserves Freehand's original punctuation-on, verbatim defaults.
// Asset-dependent features are requests, not assertions of server availability.
type NeMoOptions struct {
	DisablePunctuation      bool `json:"disablePunctuation"`
	Normalize               bool `json:"normalize"`
	ProfanityFilter         bool `json:"profanityFilter"`
	EndpointingMilliseconds int  `json:"endpointingMilliseconds"`
}

func (o NeMoOptions) Validate() error {
	if o.EndpointingMilliseconds != 0 && (o.EndpointingMilliseconds < 100 || o.EndpointingMilliseconds > 10000) {
		return errors.New("NeMo endpointing must use the server default (0) or 100–10000 milliseconds")
	}
	return nil
}
