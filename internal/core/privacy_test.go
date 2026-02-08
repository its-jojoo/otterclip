package core

import "testing"

func TestPrivacyFilterSubstring(t *testing.T) {
	pf, err := NewPrivacyFilter([]string{"token=", "password="}, false)
	if err != nil {
		t.Fatal(err)
	}

	if !pf.ShouldIgnore("my token=abc123") {
		t.Fatalf("expected ignore")
	}
	if pf.ShouldIgnore("just some harmless text") {
		t.Fatalf("did not expect ignore")
	}
}

func TestPrivacyFilterRegex(t *testing.T) {
	pf, err := NewPrivacyFilter([]string{`(?i)authorization:\s*bearer\s+\S+`}, true)
	if err != nil {
		t.Fatal(err)
	}

	if !pf.ShouldIgnore("Authorization: Bearer abc.def.ghi") {
		t.Fatalf("expected ignore")
	}
}
