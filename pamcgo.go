package main

// #cgo CFLAGS: -g -Wall
// #cgo LDFLAGS: converse.o -lpam
// #define PAM_STATIC
// #include <security/pam_modules.h>
// #include <stdlib.h>
// #include "converse/converse.h"
import "C"
import (
	"unsafe"
)

type pam struct {
	pamh *C.struct_pam_handle
}

func (p pam) GetChallengeResponse(user, prompt string) string {
	var conv unsafe.Pointer
	var enteredPwd string

	if C.pam_get_item(p.pamh, C.PAM_CONV, &conv) == C.PAM_SUCCESS {
		cPrompt := C.CString(prompt)
		defer C.free(unsafe.Pointer(cPrompt))
		cPwd := C.pam_get_challenge_response((*C.struct_pam_conv)(conv), C.PAM_PROMPT_ECHO_OFF, cPrompt)
		enteredPwd = C.GoString(cPwd)
	}

	return enteredPwd
}

func (p pam) GetUser() string {
	var user string
	var userOutput *C.char

	if ret := C.pam_get_user(p.pamh, &userOutput, (*C.char)(C.NULL)); ret == C.PAM_SUCCESS {
		user = C.GoString(userOutput)
	}

	return user
}

func newPAM(pamh *C.struct_pam_handle) pam {
	return pam{pamh}
}

// https://github.com/golang/go/wiki/cgo#Turning_C_arrays_into_Go_slices
func goStrings(argc int, argv **C.char) []string {
	if argc == 0 {
		return []string{}
	}

	length := int(argc)
	tmpslice := (*[1 << 30]*C.char)(unsafe.Pointer(argv))[:length:length]
	gostrings := make([]string, length)

	for i, s := range tmpslice {
		gostrings[i] = C.GoString(s)
	}

	return gostrings
}

//export pam_sm_setcred
func pam_sm_setcred(pamh *C.struct_pam_handle, flags, cArgc C.int, cArgs **C.char) C.int {
	return C.PAM_SUCCESS
}

//export pam_sm_authenticate
func pam_sm_authenticate(pamh *C.struct_pam_handle, flags, cArgc C.int, cArgs **C.char) C.int {
	pamService := newPAM(pamh)

	args := []string{}
	if cArgc > 0 {
		args = goStrings(int(cArgc), cArgs)
	}

	var retval C.int

	switch authenticate(pamService, int(flags), args) {
	case pamSuccess:
		retval = C.PAM_SUCCESS
	case pamCredUnavail:
		retval = C.PAM_CRED_UNAVAIL
	default:
		retval = C.PAM_AUTH_ERR
	}

	return retval
}

func main() {
}
