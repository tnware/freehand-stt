#ifndef FREEHAND_AUDIO_PERMISSION_H
#define FREEHAND_AUDIO_PERMISSION_H
int freehand_microphone_authorization(void);
// Returns zero instead of invoking TCC when the app lacks its usage string.
int freehand_microphone_request(void);
#endif
