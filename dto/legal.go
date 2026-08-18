package dto

const CurrentLegalPolicyVersion = "2026-08-18"

func HasCurrentLegalConsent(acceptedTerms bool, termsVersion string, acceptedPrivacy bool, privacyVersion string) bool {
	return acceptedTerms && acceptedPrivacy &&
		termsVersion == CurrentLegalPolicyVersion &&
		privacyVersion == CurrentLegalPolicyVersion
}
