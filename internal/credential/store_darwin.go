//go:build darwin && cgo

package credential

/*
#cgo CFLAGS: -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework Security -framework CoreFoundation -framework LocalAuthentication -framework Foundation -mmacosx-version-min=13.0
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <string.h>

// Returns a retained, non-interactive LAContext without evaluating any policy.
extern CFTypeRef credentialAuthenticationContext(void);

// Queries never prompt: a locked/denied keychain becomes a recoverable error.
static CFMutableDictionaryRef credentialQuery(const char *service, int sn, const char *account, int an) {
 CFMutableDictionaryRef q = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 CFStringRef s = CFStringCreateWithBytes(NULL, (const UInt8 *)service, sn, kCFStringEncodingUTF8, false);
 CFStringRef a = CFStringCreateWithBytes(NULL, (const UInt8 *)account, an, kCFStringEncodingUTF8, false);
 if (!q || !s || !a) { if(q) CFRelease(q); if(s) CFRelease(s); if(a) CFRelease(a); return NULL; }
 CFDictionarySetValue(q, kSecClass, kSecClassGenericPassword);
 CFDictionarySetValue(q, kSecAttrService, s);
 CFDictionarySetValue(q, kSecAttrAccount, a);
 CFTypeRef context = credentialAuthenticationContext();
 if (!context) { CFRelease(q); CFRelease(s); CFRelease(a); return NULL; }
 CFDictionarySetValue(q, kSecUseAuthenticationContext, context);
 CFRelease(context);
 CFRelease(s); CFRelease(a);
 return q;
}
static OSStatus credentialGet(const char *s, int sn, const char *a, int an, void *out, int capacity, int *length) {
 CFMutableDictionaryRef q = credentialQuery(s, sn, a, an);
 if (!q) return errSecAllocate;
 CFDictionarySetValue(q, kSecReturnData, kCFBooleanTrue);
 CFDictionarySetValue(q, kSecMatchLimit, kSecMatchLimitOne);
 CFTypeRef result = NULL;
 OSStatus status = SecItemCopyMatching(q, &result);
 CFRelease(q);
 if (status == errSecSuccess) {
  if (!result || CFGetTypeID(result) != CFDataGetTypeID() || CFDataGetLength((CFDataRef)result) > capacity) status = errSecParam;
  else { *length = (int)CFDataGetLength((CFDataRef)result); memcpy(out, CFDataGetBytePtr((CFDataRef)result), *length); }
 }
 if (result) CFRelease(result);
 return status;
}
static OSStatus credentialSet(const char *s, int sn, const char *a, int an, const void *value, int n) {
 CFMutableDictionaryRef q = credentialQuery(s, sn, a, an);
 if (!q) return errSecAllocate;
 CFDataRef data = CFDataCreate(NULL, value, n);
 CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 if (!data || !attrs) { if(data) CFRelease(data); if(attrs) CFRelease(attrs); CFRelease(q); return errSecAllocate; }
 CFDictionarySetValue(attrs, kSecValueData, data);
 OSStatus status = SecItemUpdate(q, attrs);
 if (status == errSecItemNotFound) {
  CFDictionarySetValue(q, kSecValueData, data);
  status = SecItemAdd(q, NULL);
  // Another process may have created the same account between update and add.
  if (status == errSecDuplicateItem) { CFDictionaryRemoveValue(q, kSecValueData); status = SecItemUpdate(q, attrs); }
 }
 CFRelease(attrs); CFRelease(data); CFRelease(q);
 return status;
}
static OSStatus credentialDelete(const char *s, int sn, const char *a, int an) {
 CFMutableDictionaryRef q = credentialQuery(s, sn, a, an);
 if (!q) return errSecAllocate;
 OSStatus status = SecItemDelete(q);
 CFRelease(q);
 return status;
}
*/
import "C"

import "unsafe"

func backendGet(account string) (string, error) {
	if err := validateDarwinCredential(account, "", false); err != nil {
		return "", err
	}
	buf := make([]byte, maxDarwinCredentialBytes)
	defer clear(buf)
	var n C.int
	status := C.credentialGet((*C.char)(unsafe.Pointer(unsafe.StringData(service))), C.int(len(service)), (*C.char)(unsafe.Pointer(unsafe.StringData(account))), C.int(len(account)), unsafe.Pointer(&buf[0]), C.int(len(buf)), &n)
	if err := darwinStatus(int32(status)); err != nil {
		return "", err
	}
	return string(buf[:int(n)]), nil
}
func backendSet(account, value string) error {
	if err := validateDarwinCredential(account, value, true); err != nil {
		return err
	}
	return darwinStatus(int32(C.credentialSet((*C.char)(unsafe.Pointer(unsafe.StringData(service))), C.int(len(service)), (*C.char)(unsafe.Pointer(unsafe.StringData(account))), C.int(len(account)), unsafe.Pointer(unsafe.StringData(value)), C.int(len(value)))))
}
func backendDelete(account string) error {
	if err := validateDarwinCredential(account, "", false); err != nil {
		return err
	}
	return darwinStatus(int32(C.credentialDelete((*C.char)(unsafe.Pointer(unsafe.StringData(service))), C.int(len(service)), (*C.char)(unsafe.Pointer(unsafe.StringData(account))), C.int(len(account)))))
}
