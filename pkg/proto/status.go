package proto

import "encoding/json"

type (
	BuildStatus interface {
		String() string
	}

	buildStatus struct {
		status string
	}
)

var (
	BuildStatusUnknown    BuildStatus = buildStatus{"unknown"}
	BuildStatusProcessing BuildStatus = buildStatus{"processing"}
	BuildStatusFailure    BuildStatus = buildStatus{"failure"}
	BuildStatusSuccess    BuildStatus = buildStatus{"success"}
	BuildStatusQueued     BuildStatus = buildStatus{"queued"}
)

func (status buildStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(status.status)
}

func (status buildStatus) String() string {
	return status.status
}
