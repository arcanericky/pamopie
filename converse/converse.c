#include <security/pam_appl.h>
#include "converse.h"

// a bridge function is required to call the pam_conv.conv function
// pointer anyway, so add a little more code to handle the C struct
// arrays so they don't have to be addressed in Go
char *pam_get_challenge_response(struct pam_conv *conv, int style, char *message)
{
    char *response = "";
    struct pam_message msg[1], *pmsg[1];
    struct pam_response *resp;

    msg[0].msg_style = style;
    msg[0].msg = message;
    pmsg[0] = &msg[0];

    if (conv->conv(1, (const struct pam_message **)pmsg,
            &resp, conv->appdata_ptr) == PAM_SUCCESS)
        {
        response = resp[0].resp;
        }

    return response;
}
