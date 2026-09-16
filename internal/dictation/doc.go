// Package dictation owns the live microphone workflow and its recording state
// machine. Admission, capture, processing, status, and delivery share one recorder
// owner; their files do not introduce independent lifecycle or insertion authority.
package dictation
