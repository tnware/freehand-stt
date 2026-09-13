//go:build darwin

#import <AVFoundation/AVFoundation.h>
#include "audio_permission_darwin.h"
#include <stdatomic.h>
#include <stdbool.h>

static atomic_bool microphone_request_pending = false;

int freehand_microphone_request(void) {
    @autoreleasepool {
        id usage = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"NSMicrophoneUsageDescription"];
        if (![usage isKindOfClass:[NSString class]] || [(NSString *)usage length] == 0) {
            return 0; // AVFoundation otherwise terminates a mispackaged app.
        }
        bool expected = false;
        if (atomic_compare_exchange_strong(&microphone_request_pending, &expected, true)) {
            // AVFoundation is asynchronous; no Go pointer or waiter is retained.
            // TCC owns the visible prompt, which cannot be dismissed on cancel.
            [AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {
                (void)granted;
                atomic_store(&microphone_request_pending, false);
            }];
        }
        return 1;
    }
}

int freehand_microphone_authorization(void) {
    @autoreleasepool {
        switch ([AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio]) {
            case AVAuthorizationStatusNotDetermined: return 0;
            case AVAuthorizationStatusAuthorized: return 1;
            case AVAuthorizationStatusDenied: return 2;
            case AVAuthorizationStatusRestricted: return 3;
        }
        return 3; // Fail closed for a future status.
    }
}
