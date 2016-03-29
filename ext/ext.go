// The ext package contains sub-packages that shell out to external commands.
// All the sub-packages beneath ext must use the provided *shell.Sh instance to
// shell out, so that their interaction with the shell can be properly logged/
// reported to the user.
//
// ext itself holds functions responsible for calling these sub-packages to
// create sous domain objects.
package ext
