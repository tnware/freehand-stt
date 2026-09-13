#ifndef FH_INPUT_DARWIN_H
#define FH_INPUT_DARWIN_H
#include <stdint.h>
#include <stddef.h>
typedef struct fh_input_owner fh_input_owner;
fh_input_owner *fh_input_create(void);
void fh_input_reset_target(fh_input_owner *owner);
void fh_input_destroy(fh_input_owner *owner);
int fh_input_capture(fh_input_owner *owner, uint32_t *pid, uint64_t *started);
int fh_input_validate(fh_input_owner *owner, uint32_t pid, uint64_t started);
int fh_input_modifiers_released(void);
int fh_input_send(fh_input_owner *owner, const uint16_t *units, size_t count);
enum { FH_COPY_QUEUED, FH_COPY_RUNNING, FH_COPY_SUCCEEDED, FH_COPY_FAILED, FH_COPY_CANCELLED };
void *fh_input_copy_begin(const char *text, size_t length);
int fh_input_copy_status(void *request);
int fh_input_copy_cancel(void *request);
void fh_input_copy_release(void *request);
int fh_input_accessibility_authorized(void);
void fh_input_request_accessibility(void);
#endif
