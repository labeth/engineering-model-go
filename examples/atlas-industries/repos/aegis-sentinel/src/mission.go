// ENGMODEL-OWNER-UNIT: FU-AEG-MISSION
// ENGMODEL-CODE-DESCRIPTION: presents each perimeter track to the security operator
package src

// TRLC-LINKS: REQ-AEG-001
func PresentTrack(trackID string) string {
	return "track:" + trackID
}
