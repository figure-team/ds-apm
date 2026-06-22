package alertmanagertypes

import "testing"

func TestIncidentAnnotationNotificationBodyKey(t *testing.T) {
	if IncidentAnnotationNotificationBody != "notification_body" {
		t.Fatalf("want notification_body, got %q", IncidentAnnotationNotificationBody)
	}
}
