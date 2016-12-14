// This file was automatically generated based on the contents of *.tmpl
// If you need to update this file, change the contents of those files
// (or add new ones) and run 'go generate'

package main

import "golang.org/x/tools/godoc/vfs/mapfs"

var Templates = mapfs.New(map[string]string{
	`ssl-config.tmpl`: "[ req ]\nprompt = no\ndistinguished_name=req_distinguished_name\nx509_extensions = va_c3\nencrypt_key = no\ndefault_keyfile=testing.key\ndefault_md = sha256\n\n[ va_c3 ]\nbasicConstraints=critical,CA:true,pathlen:1\n{{range . -}}\nsubjectAltName = IP:{{.}}\n{{end}}\n[ req_distinguished_name ]\nCN=registry.test\n",
})
