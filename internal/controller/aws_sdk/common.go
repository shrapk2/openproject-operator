package awsinventory

import ()

// GetRegion chooses the best region source (spec > secret > default)
func GetRegion(specRegion, secretRegion string) string {
	if specRegion != "" {
		return specRegion
	}
	if secretRegion != "" {
		return secretRegion
	}
	return "us-east-1"
}
