// ext contains sub-packages that shell out to external commands. All the
// sub-packages beneath ext must use the provided *shell.Sh instance to shell
// out, so that their interaction with the shell can be properly logged/reported
// to the user.
package ext
