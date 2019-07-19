package proto

type (
	PackageStatus string
)

const (
	PackageStatusUnknown    PackageStatus = "unknown"
	PackageStatusProcessing PackageStatus = "processing"
	PackageStatusFailure    PackageStatus = "failure"
	PackageStatusSuccess    PackageStatus = "success"
	PackageStatusQueued     PackageStatus = "queued"
)
