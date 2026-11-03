//go:build linux && cgo

#include "keyboard_linux.h"
#include <xcb/xcb.h>
#include <xcb/xcbext.h>
#include <X11/extensions/XKBproto.h>
#include <stdlib.h>
#include <string.h>
#include <poll.h>
#include <time.h>

// Dedicated XCB connection: never install process-global Xlib error handlers.
// No text conversion, key-map mutation, synthetic input, or retained typing.
struct fh_xkeyboard { xcb_connection_t *c; xcb_window_t root; uint32_t wire[256]; unsigned mods[4], locks; int xkb_event; };
static int64_t milliseconds(void) { struct timespec t; clock_gettime(CLOCK_MONOTONIC,&t); return (int64_t)t.tv_sec*1000+t.tv_nsec/1000000; }
static void *reply(fh_xkeyboard *k, unsigned seq) {
    if (!seq || !xcb_flush(k->c)) return NULL;
    int64_t end=milliseconds()+250;
    do {
        void *r=NULL; xcb_generic_error_t *e=NULL;
        if (xcb_poll_for_reply(k->c,seq,&r,&e)) { if(e) { free(e); free(r); return NULL; } return r; }
        if(xcb_connection_has_error(k->c)) break;
        struct pollfd fd={xcb_get_file_descriptor(k->c),POLLIN,0}; poll(&fd,1,5);
    } while(milliseconds()<end);
    xcb_discard_reply(k->c,seq); return NULL;
}
static int checked(fh_xkeyboard *k, xcb_void_cookie_t cookie) {
    // Complete a bounded round-trip before request_check so it cannot wait.
    void *r=reply(k,xcb_get_input_focus(k->c).sequence);
    if(!r) return 0; free(r);
    xcb_generic_error_t *e=xcb_request_check(k->c,cookie);
    if(e) { free(e); return 0; } return 1;
}
static unsigned xkb_request(fh_xkeyboard *k, void *data, unsigned size, unsigned opcode, int isvoid) {
    // The protocol structs come from the official X.Org protocol headers.
    // QueryExtension was already resolved, avoiding lazy unbounded discovery.
    ((uint8_t*)data)[1]=opcode; ((uint16_t*)data)[1]=size/4;
    struct iovec v[3]={{0},{0},{data,size}};
    xcb_protocol_request_t req={1,NULL,((uint8_t*)data)[0],isvoid};
    return xcb_send_request(k->c,XCB_REQUEST_RAW|XCB_REQUEST_CHECKED,v+2,&req);
}
static uint32_t wire_key(uint32_t s) {
    if(s>='a' && s<='z') return s-'a'+'A';
    if((s>='A' && s<='Z') || (s>='0' && s<='9') || s==0x20) return s;
    if(s>=0xffbe && s<=0xffd5) return 0x70+s-0xffbe;
    switch(s) {
    case 0xff1b:return 0x1b;
    case 0xffe1:case 0xffe2:return 0x10;
    case 0xffe3:case 0xffe4:return 0x11;
    case 0xffe9:case 0xffea:return 0x12;
    case 0xffeb:case 0xffec:return 0x5b;
    default:return 0xffff;
    }
}
fh_xkeyboard *fh_xopen(void) {
    fh_xkeyboard *k=calloc(1,sizeof(*k)); if(!k)return NULL;
    int screen=0; k->c=xcb_connect(NULL,&screen);
    if(!k->c || xcb_connection_has_error(k->c))goto fail;
    const xcb_setup_t *setup=xcb_get_setup(k->c);
    xcb_screen_iterator_t it=xcb_setup_roots_iterator(setup);
    while(screen-- && it.rem)xcb_screen_next(&it);
    if(!it.rem)goto fail; k->root=it.data->root;
    xcb_query_extension_reply_t *ext=reply(k,xcb_query_extension(k->c,9,"XKEYBOARD").sequence);
    if(!ext)goto fail;
    int opcode=ext->major_opcode; k->xkb_event=ext->first_event; int present=ext->present; free(ext);
    if(!present)goto fail;
    xkbUseExtensionReq use={0}; use.reqType=opcode; use.wantedMajor=1; use.wantedMinor=0;
    xkbUseExtensionReply *ur=reply(k,xkb_request(k,&use,sizeof(use),X_kbUseExtension,0));
    if(!ur)goto fail; int supported=ur->supported; free(ur); if(!supported)goto fail;
    xkbPerClientFlagsReq flags={0}; flags.reqType=opcode; flags.deviceSpec=XkbUseCoreKbd;
    flags.change=XkbPCF_DetectableAutoRepeatMask; flags.value=flags.change;
    xkbPerClientFlagsReply *fr=reply(k,xkb_request(k,&flags,sizeof(flags),X_kbPerClientFlags,0));
    if(!fr)goto fail; supported=(fr->supported & fr->value & XkbPCF_DetectableAutoRepeatMask)!=0; free(fr); if(!supported)goto fail;
    xkbSelectEventsReq sel={0}; sel.reqType=opcode; sel.deviceSpec=XkbUseCoreKbd;
    sel.affectWhich=sel.selectAll=XkbNewKeyboardNotifyMask|XkbMapNotifyMask;
    sel.affectMap=sel.map=XkbAllMapComponentsMask;
    xcb_void_cookie_t cookie={xkb_request(k,&sel,sizeof(sel),X_kbSelectEvents,1)};
    if(!checked(k,cookie))goto fail;
    unsigned min=setup->min_keycode, count=setup->max_keycode-min+1;
    xcb_get_keyboard_mapping_reply_t *map=reply(k,xcb_get_keyboard_mapping(k->c,min,count).sequence);
    if(!map || !map->keysyms_per_keycode) { free(map); goto fail; }
    xcb_keysym_t *syms=xcb_get_keyboard_mapping_keysyms(map); unsigned stride=map->keysyms_per_keycode;
    for(unsigned i=0;i<count;i++) k->wire[min+i]=wire_key(syms[i*stride]);
    xcb_get_modifier_mapping_reply_t *mods=reply(k,xcb_get_modifier_mapping(k->c).sequence);
    if(!mods) { free(map); goto fail; }
    uint8_t *codes=xcb_get_modifier_mapping_keycodes(mods);
    // Discover modifiers/lock bits, do not assume NumLock is Mod2 or Alt Mod1.
    for(unsigned bit=0;bit<8;bit++)for(unsigned j=0;j<mods->keycodes_per_modifier;j++) {
        unsigned code=codes[bit*mods->keycodes_per_modifier+j]; if(code<min || code>=min+count)continue;
        uint32_t s=syms[(code-min)*stride];
        switch(wire_key(s)) {
        case 0x11:k->mods[0]|=1u<<bit;break;
        case 0x12:k->mods[1]|=1u<<bit;break;
        case 0x10:k->mods[2]|=1u<<bit;break;
        case 0x5b:k->mods[3]|=1u<<bit;break;
        default:if(s==0xffe5 || s==0xff7f || s==0xff14)k->locks|=1u<<bit;break;
        }
    }
    free(mods); free(map);
    // Ambiguous/overlapping mappings cannot safely represent a saved chord.
    unsigned seen=k->locks;
    for(int i=0;i<4;i++) { unsigned m=k->mods[i]; if(!m || (m&(m-1)) || (seen&m))goto fail; seen|=m; }
    return k;
fail: fh_xclose(k); return NULL;
}
void fh_xclose(fh_xkeyboard *k) { if(k) { if(k->c)xcb_disconnect(k->c); free(k); } }
uint32_t fh_xwire(fh_xkeyboard *k,unsigned code) { return code<256?k->wire[code]:0xffff; }
unsigned fh_xmod(fh_xkeyboard *k,unsigned mod) { return mod<4?k->mods[mod]:0; }
unsigned fh_xcode(fh_xkeyboard *k,uint32_t wire) {
    unsigned found=0;
    for(unsigned i=8;i<256;i++)if(k->wire[i]==wire) { if(found)return 0; found=i; }
    return found;
}
int fh_xgrab(fh_xkeyboard *k,unsigned code,unsigned mask) {
    for(unsigned bits=0;bits<256;bits++)if((bits&~k->locks)==0) {
        if(!checked(k,xcb_grab_key_checked(k->c,0,k->root,mask|bits,code,XCB_GRAB_MODE_ASYNC,XCB_GRAB_MODE_ASYNC)))return 0;
    }
    return 1;
}
int fh_xcapture(fh_xkeyboard *k) {
    xcb_grab_keyboard_reply_t *r=reply(k,xcb_grab_keyboard(k->c,0,k->root,XCB_CURRENT_TIME,XCB_GRAB_MODE_ASYNC,XCB_GRAB_MODE_ASYNC).sequence);
    if(!r)return 0; int ok=r->status==XCB_GRAB_STATUS_SUCCESS;free(r);return ok;
}
void fh_xungrab_active(fh_xkeyboard *k) { xcb_ungrab_keyboard(k->c,XCB_CURRENT_TIME);xcb_flush(k->c); }
int fh_xreleased(fh_xkeyboard *k) {
    xcb_query_keymap_reply_t *r=reply(k,xcb_query_keymap(k->c).sequence);if(!r)return -1;
    int released=1;for(unsigned i=0;i<32;i++)if(r->keys[i])released=0;free(r);return released;
}
int fh_xread(fh_xkeyboard *k,fh_xevent *out) {
    memset(out,0,sizeof(*out));
    if(xcb_connection_has_error(k->c)) { out->lost=1; return 1; }
    xcb_generic_event_t *ev=xcb_poll_for_event(k->c);if(!ev)return 0;
    unsigned type=ev->response_type&0x7f;
    if(type==0 || type==XCB_MAPPING_NOTIFY || type==(unsigned)k->xkb_event)out->lost=1;
    else if((type==XCB_KEY_PRESS || type==XCB_KEY_RELEASE) && !(ev->response_type&0x80)) {
        xcb_key_press_event_t *e=(xcb_key_press_event_t*)ev;
        out->code=e->detail;out->state=e->state;out->down=type==XCB_KEY_PRESS;
    }
    free(ev);return 1;
}

int fh_xheld(fh_xkeyboard *k,uint32_t wire) {
    unsigned code=fh_xcode(k,wire);if(!code)return -1;
    xcb_query_keymap_reply_t *r=reply(k,xcb_query_keymap(k->c).sequence);if(!r)return -1;
    int held=(r->keys[code/8] & (1u<<(code%8)))!=0;free(r);return held;
}
