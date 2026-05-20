package events

const (
	SubjectPermitCreated            = "fieldstone.permits.permit.created"
	SubjectPermitStatusChanged      = "fieldstone.permits.permit.status_changed"
	SubjectInspectionScheduled      = "fieldstone.permits.inspection.scheduled"
	SubjectServiceRequestCreated    = "fieldstone.requests.service_request.created"
	SubjectServiceRequestAssigned   = "fieldstone.requests.service_request.assigned"
	SubjectServiceRequestClosed     = "fieldstone.requests.service_request.closed"
	SubjectFOIARequestCreated       = "fieldstone.records.foia_request.created"
	SubjectFOIARequestStatusChanged = "fieldstone.records.foia_request.status_changed"
	SubjectUserProvisioned          = "fieldstone.identity.user.provisioned"
	SubjectFOIADeadlineExceeded     = "fieldstone.records.foia_request.deadline_exceeded"
	SubjectAll                      = "fieldstone.>"
)
