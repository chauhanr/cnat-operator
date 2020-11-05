package controllers

import "testing"

func TestTimeUntil(t *testing.T) {
	schedule := "2020-10-12T15:04:00Z"
	d, err := timeUntilSchedule(schedule)
	if err != nil {
		t.Fatalf("Unexpected Error %s\n", err)
	}
	if d >= 0 {
		t.Errorf("Expected to get d greater than 0 but got %d\n", d)
	}

}
