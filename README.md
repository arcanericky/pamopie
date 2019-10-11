
# PAM OPIE Module

[![Build Status](https://travis-ci.com/arcanericky/pamopie.svg?branch=master)](https://travis-ci.com/arcanericky/pamopie)
[![codecov](https://codecov.io/gh/arcanericky/pamopie/branch/master/graph/badge.svg)](https://codecov.io/gh/arcanericky/pamopie)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

A PAM OPIE module, but written in Go! _S/KEY authentication like it's 1996_

## Quick Start for Debian and SSH

1. Install the `pam_opie.so` module

```
$ cp pam_opie.so /lib/x86_64-linux-gnu/security
```

2. Enable the module by adding the following line to `/etc/pam.d/sshd`, above the line reading `@include common-auth`

```
auth [success=done cred_unavail=ignore auth_err=die] pam_opie.so config=/etc/opie.json
```

3. Enable SSH Challenge Responses in `/etc/ssh/sshd_config` with the line

```
ChallengeResponseAuthentication yes
```

4. Restart the SSH server

```
$ systemctl restart sshd
```

5. Configure an OPIE user

```
cat > /etc/opie.json <<EOF
{
  "users": [
    {
      "name": "exampleuser",
      "passphrase": "examplepassphrase"
    }
  ]
}
EOF
```

6. Set configuration file permissions

```
$ chmod 600 /etc/opie.json
```

7. Get the OPIE prompt

```
$ ssh exampleuser@localhost
otp-md5 796 3dxgew ext
Password:
```

8. Generate and enter the OPIE challenge response into the above prompt by using [an opiekey utility](https://github.com/arcanericky/opiekey)

```
$ opiekey 796 3dxgew examplepassphrase
LIME WRIT CHOU LOVE CUE BURL
```

### This OPIE Implementation vs Traditional OPIE

This OPIE implementation is not drop-in compatible with traditional OPIE implementations. Key differences are:

#### Limited challenge response entry

Only MD5 and word (no hex) responses are currently supported.

#### No utility for the user to change their passphrase

User configuration must be done by a user that has permissions to the OPIE configuration file.

#### The configuration file does not contain state

There is no sequence number used for counting down. Sequence numbers are randomly generated when the login prompt is given.

#### Challenge seed is random

It does not follow the traditional format. It is random (but limited to alphanumeric characters) and the length is configurable.

#### A json formatted configuration file

The traditional format is line based, per user, and maintains state by counting down the sequence number. Arguably, editing the json format may be more complicated, but it is a standardized format that makes it easy to enforce structure.

These differences allow for a simpler code base by eliminating race conditions which can happen with traditional OPIE when users change their configuration (two users, or even the same user changing the configuration at the same time), and when two OPIE configured users (different users or even the same user) login at the same time, thus counting down the sequence number.

### Installation

These instructions are only an example for use on a Debian-based system and enabling this OPIE module for logins via SSH.

Install the PAM OPIE module in the default PAM module directory. This is distribution specific. For Debian, this is `/lib/x86_64-linux-gnu/security`. Other locations might be `/usr/lib/security` or `/lib/security`.
```
$ cp pam_opie.so /lib/x86_64-linux-gnu/security
```

Enable Challenge/Response Authentication for the SSH server by enabling the line `ChallengeResponseAuthentication yes` in the `sshd_config` file. For Debian, this file is at `/etc/ssh/sshd_config`. Restart the SSH server after this modification. For Debian this is done with `systemctl restart sshd`.

Configure PAM to use the OPIE module by adding it to one of the PAM config files. These files are generally in `/etc/pam.d`. Another location might be `/etc/pam.conf`. To configure this on a Debian system for the SSH server, edit `/etc/pam.d/sshd` and place the following line anywhere before `@include common-auth`. PAM configuration can get complicated and I'm no expert at it, but this is what works for me. Note there is no such thing as a PAM service that must be restarted. The following line is a good one to start with:

```
auth [success=done cred_unavail=ignore auth_err=die] pam_opie.so config=/etc/opie.json
```

The configuration line has a section with `config=/etc/opie.json`. This configures the PAM OPIE module to use `/etc/opie.json` for configuration. This file can be named anything or placed anywhere as long as it has closed permissions (read/write / 0600 only by owner) and configured properly with this config line.

Note that most traditional OPIE configurations are stored at `/etc/opiekeys`. This OPIE implementation uses a json configuration file and does not follow the traditional, line-based format.

### OPIE Configuration File

The configuration file consists of two main sections. The `defaults` and the `users`. To enable OPIE functionality, at least a default or user `passphrase` and at least one `user` must be configured, making a minimal configuration file being:

```
{
  "users": [
    {
      "name": "exampleuser",
      "passphrase": "examplepassphrase"
    }
  ]
}
```

If not specified in the configuration file, the `maxseq` is defaulted to `499`, `retries` to `1` and `seedLen` to `6`. These fields can be specified in the `defaults` section and are applied to all users. Another example:

```
{
  "defaults":
    {
      "maxseq": 999,
      "passphrase": "examplepassphrase",
      "retries": 2,
      "seedlen": 7
    },
  "users": [
    {
      "name": "exampleuser"
    }
  ]
}
```

However, if a user entry contains these fields, it will override those in the `defaults`. For example:

```
{
  "defaults": {
    "maxseq": 999,
    "passphrase": "examplepassphrase",
    "retries": 2,
    "seedlen": 7
  },
  "users": [
    {
      "name": "exampleuser",
      "maxseq": 1313,
      "passphrase": "mypersonalpassphrase",
      "retries": 3,
      "seedlen": 9
    }
  ]
}
```

Not using a `defaults` section is also a valid usage. For example:

```
{
  "users": [
    {
      "name": "exampleuser",
      "maxseq": 1313,
      "passphrase": "mypersonalpassphrase",
      "retries": 3,
      "seedlen": 9
    }
  ]
}
```

High level configuration errors are logged using syslog in `/var/log/auth.log`. Using `jq` may be useful for validating the json format using a command such as `cat /etc/opie.conf | jq '.'`

### Validation and Testing

Once configured as above for Debian and SSH, logins for OPIE-enabled users should yield an OPIE challenge. For example:

```
$ ssh exampleuser@localhost
otp-md5 796 3dxgew ext
Password: 
```

The key elements of the OPIE challenge above are the sequence number `796` and the seed of `3dxgew`. Generate the challenge response using an [opiekey utility](https://github.com/arcanericky/opiekey) and entering the sequence number, seed, and passphrase. For example:

```
$ opiekey 796 3dxgew examplepassphrase
LIME WRIT CHOU LOVE CUE BURL
```

Entering the generated challenge response above (`LIME WRIT CHOU LOVE CUE BURL`) using copy/paste or by hand should yield a successful login.

### Inspiration

I've wanted to take on a project that uses [Go cgo](https://golang.org/cmd/cgo/) and this project which creates a shared object that is called from existing C code, where the shared object makes calls back into C satisfies my curiosity to make these calls in both directions. OPIE is also implemented on a few servers for my professional job, but the traditional OPIE implementation is less than ideal for our usage. This implementation contains some experimental variations to resolve some of those shortcomings and being written mostly in Go, is much easier to maintain than the original C implementation.