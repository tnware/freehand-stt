//go:build darwin && cgo

#import <CoreFoundation/CoreFoundation.h>
#import <LocalAuthentication/LocalAuthentication.h>

// Ownership transfers to the C caller. Creating a context does not authenticate;
// refusing interaction makes locked/denied items recoverable errors, not prompts.
CFTypeRef credentialAuthenticationContext(void) {
    LAContext *context = [[LAContext alloc] init];
    context.interactionNotAllowed = YES;
    return (CFTypeRef)context;
}
