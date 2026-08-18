package dto

import "testing"

func TestHasCurrentLegalConsent(t *testing.T) {
	if !HasCurrentLegalConsent(true, CurrentLegalPolicyVersion, true, CurrentLegalPolicyVersion) {
		t.Fatal("current terms and privacy consent should be accepted")
	}
	for _, test := range []struct {
		name            string
		acceptedTerms   bool
		termsVersion    string
		acceptedPrivacy bool
		privacyVersion  string
	}{
		{name: "terms not accepted", termsVersion: CurrentLegalPolicyVersion, acceptedPrivacy: true, privacyVersion: CurrentLegalPolicyVersion},
		{name: "privacy not accepted", acceptedTerms: true, termsVersion: CurrentLegalPolicyVersion, privacyVersion: CurrentLegalPolicyVersion},
		{name: "stale terms", acceptedTerms: true, termsVersion: "2026-01-01", acceptedPrivacy: true, privacyVersion: CurrentLegalPolicyVersion},
		{name: "stale privacy", acceptedTerms: true, termsVersion: CurrentLegalPolicyVersion, acceptedPrivacy: true, privacyVersion: "2026-01-01"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if HasCurrentLegalConsent(test.acceptedTerms, test.termsVersion, test.acceptedPrivacy, test.privacyVersion) {
				t.Fatal("invalid consent should be rejected")
			}
		})
	}
}
